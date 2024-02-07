package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type Deployment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func GetDeployment(client graphql.Client, orgID string, id string) (*Deployment, error) {
	response, err := getDeploymentById(context.Background(), client, orgID, id)

	return response.Deployment.toDeployment(), err
}

func (d *getDeploymentByIdDeployment) toDeployment() *Deployment {
	return &Deployment{
		ID:     d.Id,
		Status: d.Status,
	}
}

func DeployPackage(client graphql.Client, orgID, targetID, manifestID, message string) (*Deployment, error) {
	ctx := context.Background()
	response, err := deployPackage(ctx, client, orgID, targetID, manifestID, message)

	if err != nil {
		return nil, err
	}

	if response.DeployPackage.Successful {
		return response.DeployPackage.Result.toDeployment(), nil
	}

	return nil, NewMutationError("failed to deploy package", response.DeployPackage.Messages)
}

func (d *deployPackageDeployPackageDeploymentPayloadResultDeployment) toDeployment() *Deployment {
	return &Deployment{
		ID: d.Id,
	}
}
