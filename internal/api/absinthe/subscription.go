package absinthe

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Subscription is an open Absinthe subscription. Iterate Data until it closes,
// then check Err() for any failure cause (nil means a clean termination).
type Subscription struct {
	// ID is the subscriptionId returned by Absinthe (also the Phoenix topic on
	// which subsequent subscription:data frames arrive).
	ID string

	// Data yields the raw `data` payload of each subscription:data frame (the
	// inner contents of `result.data`, with no envelope). Closed when the
	// socket dies, the subscription is closed, or the server completes.
	Data <-chan json.RawMessage

	socket   *Socket
	closeOne sync.Once
}

// Subscribe pushes a GraphQL subscription document on the Absinthe control
// channel and registers the resulting subscriptionId for routing.
func (s *Socket) Subscribe(ctx context.Context, query string, variables map[string]any) (*Subscription, error) {
	docPayload := struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables,omitempty"`
	}{Query: query, Variables: variables}
	body, err := json.Marshal(docPayload)
	if err != nil {
		return nil, fmt.Errorf("absinthe subscribe: marshal doc: %w", err)
	}

	ref := s.refID()
	resp, err := s.push(ctx, &s.joinRef, ref, controlTopic, "doc", body)
	if err != nil {
		return nil, err
	}
	if resp.status != "ok" {
		return nil, fmt.Errorf("absinthe subscribe: status=%s response=%s", resp.status, string(resp.response))
	}

	var subResp struct {
		SubscriptionID string `json:"subscriptionId"`
	}
	if err := json.Unmarshal(resp.response, &subResp); err != nil {
		return nil, fmt.Errorf("absinthe subscribe: decode subscriptionId: %w", err)
	}
	if subResp.SubscriptionID == "" {
		return nil, fmt.Errorf("absinthe subscribe: server returned no subscriptionId (response=%s)", string(resp.response))
	}

	raw := s.registerSubscription(subResp.SubscriptionID)
	out := make(chan json.RawMessage, cap(raw))

	go func() {
		defer close(out)
		for payload := range raw {
			data, ok := extractData(payload)
			if !ok {
				continue
			}
			select {
			case out <- data:
			case <-s.done:
				return
			}
		}
	}()

	return &Subscription{
		ID:     subResp.SubscriptionID,
		Data:   out,
		socket: s,
	}, nil
}

// Close unsubscribes on the server side and stops routing data to this
// Subscription's channel. Safe to call multiple times and from multiple
// goroutines.
func (sub *Subscription) Close() error {
	sub.closeOne.Do(func() {
		// Best-effort unsubscribe; ignore errors (the server may already have
		// dropped the sub or the socket may be closing).
		body, _ := json.Marshal(struct {
			SubscriptionID string `json:"subscriptionId"`
		}{SubscriptionID: sub.ID})
		ctx, cancel := context.WithTimeout(context.Background(), replyTimeout)
		defer cancel()
		ref := sub.socket.refID()
		_, _ = sub.socket.push(ctx, &sub.socket.joinRef, ref, controlTopic, "unsubscribe", body)

		sub.socket.unregisterSubscription(sub.ID)
	})
	return nil
}

// Err returns the error that caused the underlying socket to close, if any.
func (sub *Subscription) Err() error {
	return sub.socket.Err()
}

// extractData unwraps an Absinthe subscription:data payload to its inner data
// object. Returns false if the payload is malformed or has only errors.
func extractData(payload json.RawMessage) (json.RawMessage, bool) {
	var env struct {
		Result struct {
			Data   json.RawMessage `json:"data"`
			Errors json.RawMessage `json:"errors"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, false
	}
	if len(env.Result.Data) == 0 || string(env.Result.Data) == "null" {
		return nil, false
	}
	return env.Result.Data, true
}
