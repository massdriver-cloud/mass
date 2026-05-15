package environment_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// stubFollowAPI is an in-test stub for environment.FollowAPI. Each test
// pushes events on `events`; the channel is closed (or ctx cancelled) to
// signal end-of-stream. TailLogs serves canned log text per deployment.
type stubFollowAPI struct {
	mu sync.Mutex

	instances []types.Instance

	events chan types.Event

	// instance ID returned by GetDeployment, keyed by deployment ID.
	depToInstance map[string]string
	depGetErr     error

	logs   map[string]string
	logErr map[string]error
}

func newStubFollowAPI() *stubFollowAPI {
	return &stubFollowAPI{
		events:        make(chan types.Event, 32),
		depToInstance: map[string]string{},
		logs:          map[string]string{},
		logErr:        map[string]error{},
	}
}

func (s *stubFollowAPI) ListInstances(_ context.Context, _ string) ([]types.Instance, error) {
	return s.instances, nil
}

func (s *stubFollowAPI) StreamEnvironmentEvents(_ context.Context, _ string) (<-chan types.Event, error) {
	return s.events, nil
}

func (s *stubFollowAPI) GetDeployment(_ context.Context, id string) (*types.Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.depGetErr != nil {
		return nil, s.depGetErr
	}
	return &types.Deployment{
		ID:       id,
		Instance: &types.Instance{ID: s.depToInstance[id]},
	}, nil
}

func (s *stubFollowAPI) TailLogs(_ context.Context, deploymentID string, w io.Writer) error {
	s.mu.Lock()
	logErr := s.logErr[deploymentID]
	logText := s.logs[deploymentID]
	s.mu.Unlock()
	if logErr != nil {
		return logErr
	}
	_, err := io.WriteString(w, logText)
	return err
}

func TestFollowEnvironment_PrefixesLinesWithInstanceID(t *testing.T) {
	environment.FollowQuietWindow = 100 * time.Millisecond                 //nolint:reassign // intentionally shortened in tests
	t.Cleanup(func() { environment.FollowQuietWindow = 30 * time.Second }) //nolint:reassign // restore default

	api := newStubFollowAPI()
	api.instances = []types.Instance{
		{ID: "ecomm-prod-db"},    // 13 chars
		{ID: "ecomm-prod-mysql"}, // 16 chars — sets pad width
	}
	api.depToInstance["dep-db-1"] = "ecomm-prod-db"
	api.depToInstance["dep-app-1"] = "ecomm-prod-mysql"
	api.logs["dep-db-1"] = "applying db schema\nmigrations done\n"
	api.logs["dep-app-1"] = "starting mysql\nready\n"

	// Fire a RUNNING then COMPLETED event for each deployment.
	api.events <- &types.DeploymentEvent{Deployment: types.Deployment{ID: "dep-db-1", Status: "RUNNING"}}
	api.events <- &types.DeploymentEvent{Deployment: types.Deployment{ID: "dep-app-1", Status: "RUNNING"}}
	api.events <- &types.DeploymentEvent{Deployment: types.Deployment{ID: "dep-db-1", Status: "COMPLETED"}}
	api.events <- &types.DeploymentEvent{Deployment: types.Deployment{ID: "dep-app-1", Status: "COMPLETED"}}
	// Don't close `events` — the watcher exits via the quiet window after
	// both deployments have transitioned to terminal status.

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := environment.FollowEnvironment(ctx, api, "ecomm-prod", &buf); err != nil {
		t.Fatalf("FollowEnvironment: %v", err)
	}

	// Pad width = 16 (mysql id). Shorter ids get 3 spaces of padding after `]`.
	out := buf.String()
	for _, want := range []string{
		"[ecomm-prod-db]    applying db schema\n",
		"[ecomm-prod-db]    migrations done\n",
		"[ecomm-prod-mysql] starting mysql\n",
		"[ecomm-prod-mysql] ready\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing expected line %q in output:\n%s", want, out)
		}
	}
}

func TestFollowEnvironment_ExitsWhenStreamCloses(t *testing.T) {
	api := newStubFollowAPI()
	close(api.events)

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	if err := environment.FollowEnvironment(ctx, api, "ecomm-prod", &buf); err != nil {
		t.Fatalf("FollowEnvironment: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output when no deployments fire; got %q", buf.String())
	}
}

func TestFollowEnvironment_PropagatesSubscribeError(t *testing.T) {
	api := errorAPI{err: errors.New("ws handshake failed")}

	err := environment.FollowEnvironment(t.Context(), api, "ecomm-prod", io.Discard)
	if err == nil || !strings.Contains(err.Error(), "ws handshake failed") {
		t.Errorf("expected subscribe error, got %v", err)
	}
}

type errorAPI struct{ err error }

func (e errorAPI) StreamEnvironmentEvents(_ context.Context, _ string) (<-chan types.Event, error) {
	return nil, e.err
}
func (errorAPI) ListInstances(_ context.Context, _ string) ([]types.Instance, error) {
	return nil, nil
}
func (errorAPI) GetDeployment(_ context.Context, _ string) (*types.Deployment, error) {
	return nil, nil
}
func (errorAPI) TailLogs(_ context.Context, _ string, _ io.Writer) error { return nil }
