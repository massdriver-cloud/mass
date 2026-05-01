// Package instance provides command implementations for managing Massdriver instances.
package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/itchyny/gojq"
	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// DeploymentStatusSleep is the interval between deployment status polling requests.
var DeploymentStatusSleep = time.Duration(10) * time.Second

// DeploymentTimeout is the maximum duration to wait for a deployment to complete.
var DeploymentTimeout = time.Duration(5) * time.Minute

// terminalDeploymentStatuses are the deployment lifecycle states past which no
// further transitions occur.
var terminalDeploymentStatuses = map[string]struct{}{
	"COMPLETED": {},
	"FAILED":    {},
	"ABORTED":   {},
	"REJECTED":  {},
}

// DeployOptions configures how RunDeploy builds the new deployment.
type DeployOptions struct {
	// Action is the deployment action to perform. Defaults to PROVISION when empty.
	Action api.DeploymentAction
	// Message is an optional message describing the deployment.
	Message string
	// Params, when non-nil, fully replaces the instance's current configuration.
	Params map[string]any
	// PatchQueries are jq expressions applied to the resolved params prior to deploy.
	PatchQueries []string
	// LogWriter, when non-nil, switches deployment-status output for live log
	// streaming via the GraphQL subscriptions API. The status-polling chatter
	// is suppressed; only log lines and the final outcome are written.
	LogWriter io.Writer
}

// RunDeploy creates a new deployment for the named instance and polls until it
// completes or times out. When opts.LogWriter is non-nil, log batches are
// streamed to it as they arrive (and the periodic status messages are
// suppressed); otherwise the legacy status-polling output is printed to stdout.
func RunDeploy(ctx context.Context, mdClient *client.Client, name string, opts DeployOptions) (*api.Deployment, error) {
	instance, err := api.GetInstance(ctx, mdClient, name)
	if err != nil {
		return nil, err
	}

	params, err := resolveDeployParams(instance, opts.Params, opts.PatchQueries)
	if err != nil {
		return nil, err
	}

	action := opts.Action
	if action == "" {
		action = api.DeploymentActionProvision
	}

	deployment, err := api.CreateDeployment(ctx, mdClient, instance.ID, api.CreateDeploymentInput{
		Action:  action,
		Message: opts.Message,
		Params:  params,
	})
	if err != nil {
		return deployment, err
	}

	if opts.LogWriter != nil {
		return waitForDeploymentWithLogs(ctx, mdClient, deployment.ID, opts.LogWriter, DeploymentTimeout)
	}
	return waitForDeployment(ctx, mdClient, deployment.ID, DeploymentTimeout)
}

func resolveDeployParams(instance *api.Instance, params map[string]any, patchQueries []string) (map[string]any, error) {
	var result map[string]any
	if params != nil {
		interpolated := map[string]any{}
		if err := interpolateParams(params, &interpolated); err != nil {
			return nil, err
		}
		result = interpolated
	} else {
		result = instance.Params
		if result == nil {
			result = map[string]any{}
		}
	}

	for _, queryStr := range patchQueries {
		query, parseErr := gojq.Parse(queryStr)
		if parseErr != nil {
			return nil, parseErr
		}

		iter := query.Run(result)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, isErr := v.(error); isErr {
				return nil, err
			}
			patched, ok := v.(map[string]any)
			if !ok {
				return nil, errors.New("failed to cast params")
			}
			result = patched
		}
	}

	return result, nil
}

func interpolateParams(params map[string]any, interpolatedParams *map[string]any) error {
	templateData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	config := os.ExpandEnv(string(templateData))

	return json.Unmarshal([]byte(config), interpolatedParams)
}

// waitForDeployment polls the deployment until it reaches a terminal state,
// printing each status check to stdout. Returns the final Deployment on success
// or an error on FAILED/ABORTED/REJECTED.
func waitForDeployment(ctx context.Context, mdClient *client.Client, id string, timeout time.Duration) (*api.Deployment, error) {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		deployment, err := api.GetDeployment(ctx, mdClient, id)
		if err != nil {
			return nil, err
		}

		fmt.Printf("Checking deployment status for %s: %s\n", id, deployment.Status)

		if final, done, terminalErr := terminalOutcome(deployment); done {
			return final, terminalErr
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for deployment %s", id)
		}

		select {
		case <-time.After(DeploymentStatusSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// waitForDeploymentWithLogs streams the deployment's logs to w in real time
// while polling the deployment status silently in the background. Returns once
// the deployment reaches a terminal state.
func waitForDeploymentWithLogs(ctx context.Context, mdClient *client.Client, id string, w io.Writer, timeout time.Duration) (*api.Deployment, error) {
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	logs, closeStream, err := api.SubscribeDeploymentLogs(streamCtx, mdClient, id)
	if err != nil {
		// Streaming setup failed (e.g. non-PAT auth). Fall back to status polling
		// so the user still gets *some* signal rather than an opaque failure.
		fmt.Fprintf(w, "warning: log streaming unavailable: %v\nfalling back to status polling\n", err)
		return waitForDeployment(ctx, mdClient, id, timeout)
	}
	defer closeStream()

	// Goroutine: print log batches as they arrive.
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		for log := range logs {
			fmt.Fprint(w, log.Message)
		}
	}()

	// Main loop: poll status silently until terminal.
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		deployment, getErr := api.GetDeployment(ctx, mdClient, id)
		if getErr != nil {
			return nil, getErr
		}

		if final, done, terminalErr := terminalOutcome(deployment); done {
			cancelStream()
			<-streamDone
			fmt.Fprintf(w, "\n%s after %ds\n", final.Status, final.ElapsedTime)
			return final, terminalErr
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for deployment %s", id)
		}

		select {
		case <-time.After(DeploymentStatusSleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// terminalOutcome reports whether a deployment is in a terminal lifecycle
// state and, if so, the *Deployment to return and any error. A "terminal"
// status that isn't COMPLETED is reported as an error.
func terminalOutcome(d *api.Deployment) (*api.Deployment, bool, error) {
	if _, terminal := terminalDeploymentStatuses[d.Status]; !terminal {
		return nil, false, nil
	}
	if d.Status == "COMPLETED" {
		return d, true, nil
	}
	return d, true, fmt.Errorf("deployment %s in status %s", d.ID, d.Status)
}
