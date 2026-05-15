package environment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// FollowQuietWindow is how long the watcher waits with no active
// deployments and no fresh DeploymentEvents before declaring the
// rollout done. The environment deploys in dependency-ordered waves,
// so we can't bail the moment one wave's deployments are terminal —
// the next wave's haven't been created yet. The window has to outlast
// the gap between waves.
var FollowQuietWindow = 30 * time.Second

// FollowAPI is the narrow SDK surface FollowEnvironment needs. Tests
// supply a stub directly; production callers use [NewFollowAPI] to
// bind a *massdriver.Client.
type FollowAPI interface {
	StreamEnvironmentEvents(ctx context.Context, environmentID string) (<-chan types.Event, error)
	GetDeployment(ctx context.Context, id string) (*types.Deployment, error)
	TailLogs(ctx context.Context, deploymentID string, w io.Writer) error
}

// NewFollowAPI returns the production [FollowAPI] backed by the SDK client.
func NewFollowAPI(c *massdriver.Client) FollowAPI { return sdkFollowAPI{c: c} }

type sdkFollowAPI struct{ c *massdriver.Client }

func (s sdkFollowAPI) StreamEnvironmentEvents(ctx context.Context, environmentID string) (<-chan types.Event, error) {
	return s.c.Environments.StreamEvents(ctx, environmentID)
}

func (s sdkFollowAPI) GetDeployment(ctx context.Context, id string) (*types.Deployment, error) {
	return s.c.Deployments.Get(ctx, id)
}

func (s sdkFollowAPI) TailLogs(ctx context.Context, deploymentID string, w io.Writer) error {
	return s.c.Deployments.TailLogs(ctx, deploymentID, w)
}

// FollowEnvironment tails logs for every deployment that fires in an
// environment-level rollout, prefixing each line with the instance's
// id so the interleaved output stays grep-friendly.
//
// Subscribes to `environmentEvents` over WebSocket; every
// `DeploymentEvent` either kicks off a [Service.TailLogs] goroutine
// for a newly-seen deployment or updates the active set so the
// watcher knows when the rollout has gone quiet.
//
// Termination: when no deployments are active and no fresh
// DeploymentEvents have arrived for [FollowQuietWindow], the watcher
// exits. The environment deploys in dependency-ordered waves, so the
// quiet window has to outlast the gap between waves.
func FollowEnvironment(ctx context.Context, api FollowAPI, envID string, w io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eventCh, err := api.StreamEnvironmentEvents(ctx, envID)
	if err != nil {
		return fmt.Errorf("subscribe to environment events for %s: %w", envID, err)
	}

	sink := newPrefixedSink(w)
	seen := map[string]struct{}{}
	active := map[string]struct{}{}
	lastActivity := time.Now()

	var tails sync.WaitGroup
	defer tails.Wait()

	check := time.NewTicker(quietWindowTick(FollowQuietWindow))
	defer check.Stop()

	for {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				return nil
			}
			depEv, isDeploy := ev.(*types.DeploymentEvent)
			if !isDeploy {
				continue
			}
			lastActivity = time.Now()
			handleDeploymentEvent(ctx, api, depEv, seen, active, sink, &tails)

		case <-check.C:
			if len(active) == 0 && time.Since(lastActivity) >= FollowQuietWindow {
				return nil
			}

		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		}
	}
}

// handleDeploymentEvent kicks off a tail goroutine the first time we
// see a deployment and tracks whether it's still active. A deployment
// that arrives already-terminal still gets its logs streamed — TailLogs
// surfaces the historical batch and returns cleanly when the deployment
// is past terminal.
func handleDeploymentEvent(
	ctx context.Context,
	api FollowAPI,
	depEv *types.DeploymentEvent,
	seen, active map[string]struct{},
	sink *prefixedSink,
	tails *sync.WaitGroup,
) {
	depID := depEv.Deployment.ID
	if depID == "" {
		return
	}

	if _, taken := seen[depID]; !taken {
		seen[depID] = struct{}{}

		// Need the instance id to label this deployment's log lines.
		// The event payload trims it for bandwidth; one round-trip per
		// new deployment is cheap.
		dep, getErr := api.GetDeployment(ctx, depID)
		if getErr != nil || dep.Instance == nil {
			return
		}
		prefixWriter := sink.For(dep.Instance.ID)

		tails.Add(1)
		go func() {
			defer tails.Done()
			_ = api.TailLogs(ctx, depID, prefixWriter)
		}()
	}

	if deployments.IsTerminal(string(depEv.Deployment.Status)) {
		delete(active, depID)
	} else {
		active[depID] = struct{}{}
	}
}

// quietWindowTick chooses how often to check the termination condition.
// Polling at the quiet window itself feels laggy; polling at 1/4 of it
// gives us a tighter "is the rollout actually idle" answer without
// burning cycles.
func quietWindowTick(window time.Duration) time.Duration {
	tick := window / 4
	if tick < 250*time.Millisecond {
		tick = 250 * time.Millisecond
	}
	return tick
}

// prefixedSink serializes interleaved writes from per-instance tail
// goroutines and tags each line with the instance id. Writers handed
// out by [prefixedSink.For] line-buffer until they see a `\n`, then
// emit a single `[id] <line>` write under the sink's mutex so two
// goroutines never scribble on top of each other.
type prefixedSink struct {
	mu sync.Mutex
	w  io.Writer
}

func newPrefixedSink(w io.Writer) *prefixedSink {
	return &prefixedSink{w: w}
}

// For returns an io.Writer that prefixes every line it writes with
// "[<id>] " and forwards to the shared writer.
func (s *prefixedSink) For(id string) io.Writer {
	return &prefixedWriter{sink: s, prefix: []byte("[" + id + "] ")}
}

type prefixedWriter struct {
	sink   *prefixedSink
	prefix []byte
	buf    bytes.Buffer
}

func (p *prefixedWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.buf.Write(b)
	if writeErr := p.flushCompleteLines(); writeErr != nil {
		return n, writeErr
	}
	return n, nil
}

// flushCompleteLines emits every fully-terminated line in the buffer
// under the sink mutex. A trailing partial line stays in the buffer
// until the next Write provides the terminating newline.
func (p *prefixedWriter) flushCompleteLines() error {
	for {
		raw := p.buf.Bytes()
		idx := bytes.IndexByte(raw, '\n')
		if idx < 0 {
			return nil
		}
		line := raw[:idx+1]
		p.sink.mu.Lock()
		if _, err := p.sink.w.Write(p.prefix); err != nil {
			p.sink.mu.Unlock()
			return err
		}
		if _, err := p.sink.w.Write(line); err != nil {
			p.sink.mu.Unlock()
			return err
		}
		p.sink.mu.Unlock()
		p.buf.Next(idx + 1)
	}
}
