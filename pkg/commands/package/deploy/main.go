package deploy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

var DeploymentStatusSleep = time.Duration(10) * time.Second
var DeploymentTimeout = time.Duration(5) * time.Minute

func Run(ctx context.Context, mdClient *client.Client, name, message string) (*api.Deployment, error) {
	pkg, err := api.GetPackageByName(ctx, mdClient, name)
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
