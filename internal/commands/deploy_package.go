package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

var DeploymentStatusSleep time.Duration = time.Duration(10) * time.Second
var DeploymentTimeout time.Duration = time.Duration(5) * time.Minute

func DeployPackage(client graphql.Client, orgID string, name string) (*api.Deployment, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)
	if err != nil {
		return nil, err
	}

	deployment, err := api.DeployPackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID)
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

	// TODO: replace w/ bubbletea (human) & zerolog (json)
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
