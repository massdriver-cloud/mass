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
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// DeploymentStatusSleep is the interval between deployment status polling requests.
var DeploymentStatusSleep = time.Duration(10) * time.Second

// DeploymentTimeout is the maximum duration to wait for a deployment to complete.
var DeploymentTimeout = time.Duration(5) * time.Minute

// DeployAPI is the narrow SDK surface RunDeploy needs. Tests supply a fake
// directly; production callers use [NewDeployAPI] to bind a *massdriver.Client.
type DeployAPI interface {
	GetInstance(ctx context.Context, id string) (*types.Instance, error)
	CreateDeployment(ctx context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error)
	GetDeployment(ctx context.Context, id string) (*types.Deployment, error)
	TailLogs(ctx context.Context, deploymentID string, w io.Writer) error
}

// NewDeployAPI returns the production [DeployAPI] backed by the SDK client.
func NewDeployAPI(c *massdriver.Client) DeployAPI { return sdkDeployAPI{c: c} }

type sdkDeployAPI struct{ c *massdriver.Client }

func (s sdkDeployAPI) GetInstance(ctx context.Context, id string) (*types.Instance, error) {
	return s.c.Instances.Get(ctx, id)
}

func (s sdkDeployAPI) CreateDeployment(ctx context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error) {
	return s.c.Deployments.Create(ctx, instanceID, in)
}

func (s sdkDeployAPI) GetDeployment(ctx context.Context, id string) (*types.Deployment, error) {
	return s.c.Deployments.Get(ctx, id)
}

func (s sdkDeployAPI) TailLogs(ctx context.Context, deploymentID string, w io.Writer) error {
	return s.c.Deployments.TailLogs(ctx, deploymentID, w)
}

// DeployOptions configures how RunDeploy builds the new deployment.
type DeployOptions struct {
	// Action is the deployment action to perform. Defaults to PROVISION when empty.
	Action deployments.Action
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
func RunDeploy(ctx context.Context, api DeployAPI, name string, opts DeployOptions) (*types.Deployment, error) {
	inst, err := api.GetInstance(ctx, name)
	if err != nil {
		return nil, err
	}

	params, err := resolveDeployParams(inst, opts.Params, opts.PatchQueries)
	if err != nil {
		return nil, err
	}

	action := opts.Action
	if action == "" {
		action = deployments.ActionProvision
	}

	deployment, err := api.CreateDeployment(ctx, inst.ID, deployments.CreateInput{
		Action:  action,
		Message: opts.Message,
		Params:  params,
	})
	if err != nil {
		return deployment, err
	}

	if opts.LogWriter != nil {
		return waitForDeploymentWithLogs(ctx, api, deployment.ID, opts.LogWriter)
	}
	return waitForDeployment(ctx, api, deployment.ID, DeploymentTimeout)
}

func resolveDeployParams(inst *types.Instance, params map[string]any, patchQueries []string) (map[string]any, error) {
	var result map[string]any
	if params != nil {
		interpolated := map[string]any{}
		if err := interpolateParams(params, &interpolated); err != nil {
			return nil, err
		}
		result = interpolated
	} else {
		result = inst.Params
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
func waitForDeployment(ctx context.Context, api DeployAPI, id string, timeout time.Duration) (*types.Deployment, error) {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		deployment, err := api.GetDeployment(ctx, id)
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

// waitForDeploymentWithLogs tails the deployment's logs to w in real time,
// returning when the deployment reaches a terminal state. If the configured
// credentials can't open the streaming websocket (basic-auth /
// service-account), the call falls back to status polling — the deploy was
// created successfully, so the user shouldn't see a non-zero exit just
// because their auth doesn't support live tailing.
func waitForDeploymentWithLogs(ctx context.Context, api DeployAPI, id string, w io.Writer) (*types.Deployment, error) {
	err := api.TailLogs(ctx, id, w)
	if errors.Is(err, deployments.ErrStreamingRequiresPAT) {
		fmt.Fprintln(os.Stderr, "warning: log streaming requires a personal access token (mds_*/md_*); falling back to status polling")
		return waitForDeployment(ctx, api, id, DeploymentTimeout)
	}
	if err != nil {
		return nil, err
	}

	// TailLogs returned cleanly, meaning the deployment is terminal. Fetch the
	// final record so we can report status + elapsed time consistently with
	// the polling path.
	final, getErr := api.GetDeployment(ctx, id)
	if getErr != nil {
		return nil, getErr
	}
	fmt.Fprintf(w, "\n%s after %ds\n", final.Status, final.ElapsedTime)
	if _, done, terminalErr := terminalOutcome(final); done {
		return final, terminalErr
	}
	return final, nil
}

// terminalOutcome reports whether a deployment is in a terminal lifecycle
// state and, if so, the *Deployment to return and any error. A "terminal"
// status that isn't COMPLETED is reported as an error.
func terminalOutcome(d *types.Deployment) (*types.Deployment, bool, error) {
	if !deployments.IsTerminal(d.Status) {
		return nil, false, nil
	}
	if d.Status == string(deployments.StatusCompleted) {
		return d, true, nil
	}
	return d, true, fmt.Errorf("deployment %s in status %s", d.ID, d.Status)
}
