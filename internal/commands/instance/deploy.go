// Package instance provides command implementations for managing Massdriver instances.
package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
}

// RunDeploy creates a new deployment for the named instance and polls until it completes or times out.
//
// When opts.Params is nil, the instance's last configuration is reused; otherwise the provided
// params replace it (with bash-style environment interpolation). PatchQueries are jq expressions
// applied to the resolved params before the deployment is created.
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

	return checkDeploymentStatus(ctx, mdClient, deployment.ID, DeploymentTimeout)
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

func checkDeploymentStatus(ctx context.Context, mdClient *client.Client, id string, timeout time.Duration) (*api.Deployment, error) {
	deployment, err := api.GetDeployment(ctx, mdClient, id)

	if err != nil {
		return nil, err
	}

	timeout -= DeploymentStatusSleep

	fmt.Printf("Checking deployment status for %s: %s\n", id, deployment.Status)

	switch deployment.Status {
	case "COMPLETED":
		return deployment, nil
	case "FAILED":
		return nil, errors.New("deployment failed")
	default:
		time.Sleep(DeploymentStatusSleep)
		return checkDeploymentStatus(ctx, mdClient, id, timeout)
	}
}
