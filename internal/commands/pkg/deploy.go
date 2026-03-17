package pkg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// DeploymentStatusSleep is the interval between deployment status polling requests.
var DeploymentStatusSleep = time.Duration(10) * time.Second

// DeploymentTimeout is the maximum duration to wait for a deployment to complete.
var DeploymentTimeout = time.Duration(5) * time.Minute

// RunDeploy deploys the named package and polls until the deployment completes or times out.
func RunDeploy(ctx context.Context, mdClient *client.Client, name, message string) (*api.Deployment, error) {
	pkg, err := api.GetPackage(ctx, mdClient, name)
	if err != nil {
		return nil, err
	}

	deployment, err := api.DeployPackage(ctx, mdClient, pkg.Environment.ID, pkg.Manifest.ID, message)
	if err != nil {
		return deployment, err
	}

	return checkDeploymentStatus(ctx, mdClient, deployment.ID, DeploymentTimeout)
}

func checkDeploymentStatus(ctx context.Context, mdClient *client.Client, id string, timeout time.Duration) (*api.Deployment, error) {
	deployment, err := api.GetDeployment(ctx, mdClient, id)

	if err != nil {
		return nil, err
	}

	timeout -= DeploymentStatusSleep

	// TODO: bubbletea status
	// API responses can updated UI...
	// 	https://github.com/Evertras/bubble-table/blob/main/examples/updates/main.go#L104
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
