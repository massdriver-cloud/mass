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
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// FollowPollInterval controls how often each instance watcher polls for new
// deployments. Exposed as a var so tests can drop it to zero.
var FollowPollInterval = 2 * time.Second

// FollowQuietWindow is how long the watcher loop waits with no new
// deployments observed before declaring the rollout done. The environment
// deploys in dependency-ordered waves, so we can't bail the moment one
// wave's deployments are terminal — the next wave's haven't been created
// yet. The window has to be long enough to cover the gap between waves.
var FollowQuietWindow = 30 * time.Second

// FollowAPI is the narrow SDK surface FollowEnvironment needs. Tests supply a
// stub directly; production callers use [NewFollowAPI] to bind a
// *massdriver.Client.
type FollowAPI interface {
	ListInstances(ctx context.Context, input instances.ListInput) ([]types.Instance, error)
	ListDeployments(ctx context.Context, input deployments.ListInput) ([]types.Deployment, error)
	TailLogs(ctx context.Context, deploymentID string, w io.Writer) error
}

// NewFollowAPI returns the production [FollowAPI] backed by the SDK client.
func NewFollowAPI(c *massdriver.Client) FollowAPI { return sdkFollowAPI{c: c} }

type sdkFollowAPI struct{ c *massdriver.Client }

func (s sdkFollowAPI) ListInstances(ctx context.Context, input instances.ListInput) ([]types.Instance, error) {
	return s.c.Instances.List(ctx, input)
}

func (s sdkFollowAPI) ListDeployments(ctx context.Context, input deployments.ListInput) ([]types.Deployment, error) {
	return s.c.Deployments.List(ctx, input)
}

func (s sdkFollowAPI) TailLogs(ctx context.Context, deploymentID string, w io.Writer) error {
	return s.c.Deployments.TailLogs(ctx, deploymentID, w)
}

// FollowEnvironment tails logs for every deployment that fires in an
// environment-level rollout, prefixing each line with the instance's id so
// the interleaved output is grep-friendly.
//
// The platform deploys in dependency-ordered waves: wave 1 instances start,
// finish, then wave 2 instances start. This watcher polls per-instance
// deployment lists every [FollowPollInterval], spawns a [Service.TailLogs]
// goroutine for each new deployment it sees, and exits when no new
// deployments have appeared for [FollowQuietWindow] and every observed
// deployment is in a terminal state.
//
// The user's hint on the design ticket was to use Absinthe subscriptions
// (`environmentEvents` / `instanceEvents`) instead of polling. The SDK's
// websocket / Absinthe machinery is in an internal package, so subscribing
// from the CLI would mean either re-porting that layer or waiting for the
// SDK to expose event subscriptions. The polling approach has a few
// seconds of discovery latency per deployment, which is acceptable
// relative to per-instance deploy times measured in minutes.
func FollowEnvironment(ctx context.Context, api FollowAPI, envID string, w io.Writer) error {
	insts, err := api.ListInstances(ctx, instances.ListInput{EnvironmentID: envID})
	if err != nil {
		return fmt.Errorf("list instances in environment %s: %w", envID, err)
	}
	if len(insts) == 0 {
		return nil
	}

	out := newPrefixedSink(w)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(insts))

	for _, inst := range insts {
		wg.Add(1)
		go func(inst types.Instance) {
			defer wg.Done()
			if watchErr := watchInstanceDeployments(ctx, api, inst, out); watchErr != nil && !errors.Is(watchErr, context.Canceled) {
				errCh <- watchErr
			}
		}(inst)
	}

	wg.Wait()
	close(errCh)

	var firstErr error
	for werr := range errCh {
		if firstErr == nil {
			firstErr = werr
		}
	}
	return firstErr
}

// watchInstanceDeployments polls for new deployments on a single instance
// and tails each one as it's created. Returns when the quiet window elapses
// with every observed deployment in a terminal state.
func watchInstanceDeployments(ctx context.Context, api FollowAPI, inst types.Instance, sink *prefixedSink) error {
	prefixWriter := sink.For(inst.ID)
	seen := map[string]struct{}{}
	var inFlight sync.WaitGroup
	var lastChange time.Time
	allTerminal := true

	for {
		deps, listErr := api.ListDeployments(ctx, deployments.ListInput{
			InstanceID: inst.ID,
			PageSize:   25,
		})
		if listErr != nil {
			return fmt.Errorf("list deployments for %s: %w", inst.ID, listErr)
		}

		anyNew := false
		anyActive := false
		for _, dep := range deps {
			if _, taken := seen[dep.ID]; !taken {
				seen[dep.ID] = struct{}{}
				anyNew = true
				lastChange = time.Now()

				inFlight.Add(1)
				go func(depID string) {
					defer inFlight.Done()
					// TailLogs blocks until the deployment hits a terminal
					// status. Errors are silenced per-deployment so one
					// bad tail doesn't take the whole follow down — the
					// final deployment list will still show the failure.
					_ = api.TailLogs(ctx, depID, prefixWriter)
				}(dep.ID)
			}
			if !deployments.IsTerminal(string(dep.Status)) {
				anyActive = true
			}
		}

		if anyNew {
			allTerminal = false
		}
		if !anyActive && !anyNew && !lastChange.IsZero() && time.Since(lastChange) >= FollowQuietWindow {
			allTerminal = true
			break
		}

		select {
		case <-time.After(FollowPollInterval):
		case <-ctx.Done():
			inFlight.Wait()
			return ctx.Err()
		}
	}

	inFlight.Wait()
	_ = allTerminal
	return nil
}

// prefixedSink serializes interleaved writes from per-instance tail
// goroutines and tags each line with the instance id. Writers handed out by
// [prefixedSink.For] line-buffer until they see a `\n`, then emit a single
// `[id] <line>` write under the sink's mutex so two goroutines never
// scribble on top of each other.
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

// flushCompleteLines emits every fully-terminated line in the buffer under
// the sink mutex. A trailing partial line stays in the buffer until the
// next Write provides the terminating newline.
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
