package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/api"
)

var DeploymentStatusSleep = time.Duration(10) * time.Second
var DeploymentTimeout = time.Duration(5) * time.Minute

func DeployPackage(client graphql.Client, orgID, name, message string) (*api.Deployment, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)
	if err != nil {
		return nil, err
	}

	deployment, err := api.DeployPackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID, message)
	if err != nil {
		return deployment, err
	}

	return checkDeploymentStatus(client, orgID, deployment.ID, DeploymentTimeout)
}

func checkDeploymentStatus(client graphql.Client, orgID string, id string, timeout time.Duration) (*api.Deployment, error) {
	deployment, err := api.GetDeployment(client, orgID, id)

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
		return checkDeploymentStatus(client, orgID, id, timeout)
	}
}
