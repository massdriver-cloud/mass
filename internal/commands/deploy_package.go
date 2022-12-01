package commands

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func DeployPackage(client graphql.Client, orgID string, name string) (api.Deployment, error) {
	deployment := api.Deployment{}

	pkg, err := api.GetPackageByName(client, orgID, name)
	if err != nil {
		return deployment, err
	}

	deployment, err = api.DeployPackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID)
	if err != nil {
		return deployment, err
	}

	// TODO: internal & loop
	deployment, err = api.GetDeployment(client, orgID, deployment.ID)

	fmt.Printf("Err3 %v", err)
	return deployment, err
}
