package api

import (
	"context"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Deployment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func GetDeployment(ctx context.Context, mdClient *client.Client, id string) (*Deployment, error) {
	response, err := getDeploymentById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id)

	return response.Deployment.toDeployment(), err
}

func (d *getDeploymentByIdDeployment) toDeployment() *Deployment {
	return &Deployment{
		ID:     d.Id,
		Status: string(d.Status),
	}
}

func DeployPackage(ctx context.Context, mdClient *client.Client, targetID, manifestID, message string) (*Deployment, error) {
	response, err := deployPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, targetID, manifestID, message)

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
