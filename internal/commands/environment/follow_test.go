package environment_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// stubFollowAPI is an in-test stub for environment.FollowAPI. Each test
// supplies instances + per-deployment log scripts; the stub serves them
// back when polled.
type stubFollowAPI struct {
	mu sync.Mutex

	instances []types.Instance
	// deployments per instance, returned in order across successive
	// ListDeployments calls. Empty means no deployments for that instance.
	depQueue map[string][][]types.Deployment
	depCalls map[string]int

	logs   map[string]string
	logErr map[string]error
}

func (s *stubFollowAPI) ListInstances(_ context.Context, _ instances.ListInput) ([]types.Instance, error) {
	return s.instances, nil
}

func (s *stubFollowAPI) ListDeployments(_ context.Context, input deployments.ListInput) ([]types.Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	queue := s.depQueue[input.InstanceID]
	call := s.depCalls[input.InstanceID]
	if call >= len(queue) {
		// Stable past the script — repeat the final entry to keep the
		// poller in steady state.
		if len(queue) == 0 {
			return nil, nil
		}
		return queue[len(queue)-1], nil
	}
	s.depCalls[input.InstanceID] = call + 1
	return queue[call], nil
}

func (s *stubFollowAPI) TailLogs(_ context.Context, deploymentID string, w io.Writer) error {
	if err := s.logErr[deploymentID]; err != nil {
		return err
	}
	_, err := io.WriteString(w, s.logs[deploymentID])
	return err
}

func TestFollowEnvironment_PrefixesLinesWithInstanceID(t *testing.T) {
	// Speed the loop up so the test isn't slow.
	environment.FollowPollInterval = 0
	environment.FollowQuietWindow = 10 * time.Millisecond
	t.Cleanup(func() {
		environment.FollowPollInterval = 2 * time.Second
		environment.FollowQuietWindow = 30 * time.Second
	})

	api := &stubFollowAPI{
		instances: []types.Instance{
			{ID: "ecomm-prod-db", Name: "db"},
			{ID: "ecomm-prod-app", Name: "app"},
		},
		depQueue: map[string][][]types.Deployment{
			"ecomm-prod-db":  {{{ID: "dep-db-1", Status: "COMPLETED"}}},
			"ecomm-prod-app": {{{ID: "dep-app-1", Status: "COMPLETED"}}},
		},
		depCalls: map[string]int{},
		logs: map[string]string{
			"dep-db-1":  "applying db schema\nmigrations done\n",
			"dep-app-1": "starting app\nready\n",
		},
		logErr: map[string]error{},
	}

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := environment.FollowEnvironment(ctx, api, "ecomm-prod", &buf); err != nil {
		t.Fatalf("FollowEnvironment: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[ecomm-prod-db] applying db schema\n") {
		t.Errorf("missing prefixed db line in:\n%s", out)
	}
	if !strings.Contains(out, "[ecomm-prod-db] migrations done\n") {
		t.Errorf("missing prefixed db line 2 in:\n%s", out)
	}
	if !strings.Contains(out, "[ecomm-prod-app] starting app\n") {
		t.Errorf("missing prefixed app line in:\n%s", out)
	}
	if !strings.Contains(out, "[ecomm-prod-app] ready\n") {
		t.Errorf("missing prefixed app line 2 in:\n%s", out)
	}
}

func TestFollowEnvironment_NoOpForEmptyEnv(t *testing.T) {
	api := &stubFollowAPI{
		instances: nil,
	}

	var buf bytes.Buffer
	if err := environment.FollowEnvironment(t.Context(), api, "ecomm-prod", &buf); err != nil {
		t.Fatalf("FollowEnvironment: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty env; got %q", buf.String())
	}
}
