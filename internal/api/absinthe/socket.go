// Package absinthe implements a thin client for GraphQL subscriptions over
// Phoenix Channels (v2 framing) as exposed by Elixir/Absinthe servers.
//
// It is intentionally generic — callers supply a GraphQL operation string and
// variables, and receive the raw `data` payload of each subscription:data frame.
// No domain types from the surrounding application leak into this package, so
// it can be lifted into the SDK alongside the gql client without changes.
package absinthe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	controlTopic      = "__absinthe__:control"
	heartbeatTopic    = "phoenix"
	heartbeatEvent    = "heartbeat"
	heartbeatInterval = 30 * time.Second
	replyTimeout      = 30 * time.Second
	socketPath        = "/api/socket/websocket"
	phoenixVsn        = "2.0.0"
)

// Socket is an open Phoenix Channels connection that has joined Absinthe's
// `__absinthe__:control` topic and can multiplex GraphQL subscriptions over a
// single WebSocket.
type Socket struct {
	conn    *websocket.Conn
	nextRef atomic.Uint64

	joinRef string

	writeMu sync.Mutex // serializes writes; gorilla requires single-writer

	mu          sync.Mutex
	pending     map[string]chan reply           // ref → reply slot
	subscribers map[string]chan json.RawMessage // subscriptionId (== Phoenix topic) → data
	closed      bool
	closeErr    error

	closeConnOne sync.Once     // gates conn.Close so user-Close is idempotent
	connCloseErr error         //   captured once for the caller
	done         chan struct{} // closed when the read loop exits
}

// reply carries a phx_reply payload back to the caller awaiting it.
type reply struct {
	status   string
	response json.RawMessage
}

// Dial opens a WebSocket to baseURL's Phoenix endpoint, joins
// `__absinthe__:control`, and starts the read/heartbeat loops.
//
// baseURL is the HTTPS API base (e.g. "https://api.example.com"). It's converted
// to wss://example.com/api/socket/websocket?token=<token>&vsn=2.0.0. The token is
// sent as a query parameter because browsers can't set headers on WebSocket
// upgrades — Phoenix UserSockets typically accept it from `connect/3`.
func Dial(ctx context.Context, baseURL, token string) (*Socket, error) {
	wsURL, err := buildWSURL(baseURL, token)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("absinthe dial: %w", err)
	}

	s := &Socket{
		conn:        conn,
		pending:     make(map[string]chan reply),
		subscribers: make(map[string]chan json.RawMessage),
		done:        make(chan struct{}),
	}
	go s.readLoop()

	if joinErr := s.join(ctx); joinErr != nil {
		_ = s.Close()
		return nil, joinErr
	}
	go s.heartbeatLoop()
	return s, nil
}

// Close terminates the WebSocket and aborts any in-flight subscriptions.
// Safe to call multiple times and from multiple goroutines.
func (s *Socket) Close() error {
	s.shutdown(nil)
	s.closeConnOne.Do(func() {
		s.connCloseErr = s.conn.Close()
	})
	return s.connCloseErr
}

// Err returns the error that caused the socket to close, if any. Returns nil
// while the socket is healthy.
func (s *Socket) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeErr
}

func (s *Socket) join(ctx context.Context) error {
	s.joinRef = s.refID()
	resp, err := s.push(ctx, &s.joinRef, s.joinRef, controlTopic, "phx_join", json.RawMessage("{}"))
	if err != nil {
		return fmt.Errorf("absinthe join: %w", err)
	}
	if resp.status != "ok" {
		return fmt.Errorf("absinthe join: status=%s response=%s", resp.status, string(resp.response))
	}
	return nil
}

// push writes a frame and (if ref is non-nil) waits for the matching phx_reply.
// joinRef is the channel's join_ref; for control-topic pushes it's the join we
// did at startup, for heartbeats it's nil.
func (s *Socket) push(ctx context.Context, joinRef *string, ref, topic, event string, payload json.RawMessage) (reply, error) {
	ch := make(chan reply, 1)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return reply{}, fmt.Errorf("absinthe socket closed: %w", s.closeErr)
	}
	s.pending[ref] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, ref)
		s.mu.Unlock()
	}()

	if err := s.writeFrame(joinRef, &ref, topic, event, payload); err != nil {
		return reply{}, err
	}

	select {
	case r := <-ch:
		return r, nil
	case <-ctx.Done():
		return reply{}, ctx.Err()
	case <-s.done:
		return reply{}, fmt.Errorf("absinthe socket closed: %w", s.Err())
	case <-time.After(replyTimeout):
		return reply{}, fmt.Errorf("absinthe push %s: timeout waiting for reply", event)
	}
}

// writeFrame sends one Phoenix v2 frame. payload may be nil to indicate {}.
func (s *Socket) writeFrame(joinRef, ref *string, topic, event string, payload json.RawMessage) error {
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	frame := []any{joinRef, ref, topic, event, payload}
	body, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteMessage(websocket.TextMessage, body)
}

func (s *Socket) refID() string {
	return strconv.FormatUint(s.nextRef.Add(1), 10)
}

func (s *Socket) readLoop() {
	defer close(s.done)
	for {
		_, body, err := s.conn.ReadMessage()
		if err != nil {
			s.shutdown(err)
			return
		}
		s.dispatch(body)
	}
}

func (s *Socket) dispatch(body []byte) {
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err != nil || len(arr) != 5 {
		// Malformed — ignore. Phoenix doesn't send anything else over WS.
		return
	}
	var (
		joinRef *string
		ref     *string
		topic   string
		event   string
	)
	_ = json.Unmarshal(arr[0], &joinRef)
	_ = json.Unmarshal(arr[1], &ref)
	if err := json.Unmarshal(arr[2], &topic); err != nil {
		return
	}
	if err := json.Unmarshal(arr[3], &event); err != nil {
		return
	}
	payload := arr[4]

	switch event {
	case "phx_reply":
		if ref == nil {
			return
		}
		var p struct {
			Status   string          `json:"status"`
			Response json.RawMessage `json:"response"`
		}
		_ = json.Unmarshal(payload, &p)
		s.mu.Lock()
		ch, ok := s.pending[*ref]
		s.mu.Unlock()
		if ok {
			ch <- reply{status: p.Status, response: p.Response}
		}
	case "subscription:data":
		s.mu.Lock()
		ch, ok := s.subscribers[topic]
		s.mu.Unlock()
		if !ok {
			return
		}
		// Payload is {"result": {"data": ..., "errors": ...}, "subscriptionId": "..."}.
		// We pass the whole payload through; the domain caller will pick what it wants.
		select {
		case ch <- payload:
		default:
			// drop on slow consumer; subscriptions:data is fire-and-forget
		}
	case "phx_error", "phx_close":
		s.shutdown(fmt.Errorf("absinthe socket received %s on topic %s: %s", event, topic, string(payload)))
	}
}

func (s *Socket) heartbeatLoop() {
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			ref := s.refID()
			// Heartbeats use a nil join_ref. Don't wait for the reply.
			_ = s.writeFrame(nil, &ref, heartbeatTopic, heartbeatEvent, nil)
		}
	}
}

func (s *Socket) shutdown(cause error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if cause != nil && !isNormalClose(cause) {
		s.closeErr = cause
	}
	pending := s.pending
	s.pending = nil
	subs := s.subscribers
	s.subscribers = nil
	s.mu.Unlock()

	for _, ch := range pending {
		close(ch)
	}
	for _, ch := range subs {
		close(ch)
	}
}

// registerSubscription wires up a topic→channel route for incoming
// subscription:data frames. Returns the channel.
func (s *Socket) registerSubscription(topic string) chan json.RawMessage {
	ch := make(chan json.RawMessage, 64)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		close(ch)
		return ch
	}
	s.subscribers[topic] = ch
	s.mu.Unlock()
	return ch
}

func (s *Socket) unregisterSubscription(topic string) {
	s.mu.Lock()
	ch, ok := s.subscribers[topic]
	if ok {
		delete(s.subscribers, topic)
	}
	s.mu.Unlock()
	if ok {
		close(ch)
	}
}

func buildWSURL(baseURL, token string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + socketPath
	q := u.Query()
	q.Set("token", token)
	q.Set("vsn", phoenixVsn)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func isNormalClose(err error) bool {
	if errors.Is(err, websocket.ErrCloseSent) {
		return true
	}
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		switch ce.Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived:
			return true
		}
	}
	return false
}
