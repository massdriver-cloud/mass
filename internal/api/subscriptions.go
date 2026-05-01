package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/massdriver-cloud/mass/internal/api/absinthe"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// deploymentLogsSubscription is the GraphQL operation pushed on the Absinthe
// control channel. Kept in lockstep with the schema's `deploymentLogs`
// subscription field — the server replies with one DeploymentLog per worker
// flush.
const deploymentLogsSubscription = `subscription deploymentLogs($organizationId: ID!, $deploymentId: ID!) {
  deploymentLogs(organizationId: $organizationId, deploymentId: $deploymentId) {
    timestamp
    message
  }
}`

// SubscribeDeploymentLogs opens an Absinthe subscription for the given
// deployment's log stream. The returned channel yields one DeploymentLog per
// server batch and is closed when ctx is cancelled, the socket dies, or the
// server completes the subscription.
//
// The returned cancel function tears down both the subscription and the
// underlying WebSocket — call it with `defer` even if you also cancel ctx.
//
// Subscriptions require PAT (bearer) authentication. Basic-auth (API_KEY in
// org:apiKey form) is rejected with an error since browsers/Phoenix sockets
// only support tokens passed as a query parameter.
func SubscribeDeploymentLogs(ctx context.Context, mdClient *client.Client, deploymentID string) (<-chan DeploymentLog, func(), error) {
	if mdClient.Config.Credentials.Method != config.AuthPAT {
		return nil, nil, errors.New("deployment log streaming requires a personal access token (set MASSDRIVER_API_KEY to a token starting with mds_/md_)")
	}

	socket, err := absinthe.Dial(ctx, mdClient.Config.URL, mdClient.Config.Credentials.Secret)
	if err != nil {
		return nil, nil, err
	}

	sub, err := socket.Subscribe(ctx, deploymentLogsSubscription, map[string]any{
		"organizationId": mdClient.Config.OrganizationID,
		"deploymentId":   deploymentID,
	})
	if err != nil {
		_ = socket.Close()
		return nil, nil, err
	}

	out := make(chan DeploymentLog, cap(sub.Data))
	go func() {
		defer close(out)
		for raw := range sub.Data {
			var env struct {
				DeploymentLogs DeploymentLog `json:"deploymentLogs"`
			}
			if err := json.Unmarshal(raw, &env); err != nil {
				// Skip malformed batches rather than tearing down the stream.
				continue
			}
			select {
			case out <- env.DeploymentLogs:
			case <-ctx.Done():
				return
			}
		}
	}()

	cancel := func() {
		_ = sub.Close()
		_ = socket.Close()
	}
	return out, cancel, nil
}
