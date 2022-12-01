package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Deployment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func GetDeployment(client graphql.Client, orgID string, id string) (Deployment, error) {
	response, err := getDeploymentById(context.Background(), client, orgID, id)

	return response.Deployment.toDeployment(), err
}

func (d *getDeploymentByIdDeployment) toDeployment() Deployment {
	return Deployment{
		ID:     d.Id,
		Status: d.Status,
	}
}

func DeployPackage(client graphql.Client, orgID string, targetID string, manifestID string) (Deployment, error) {
	ctx := context.Background()
	deployment := Deployment{}
	response, err := deployPackage(ctx, client, orgID, targetID, manifestID)

	if err != nil {
		return deployment, err
	}

	if response.DeployPackage.Successful {
		return response.DeployPackage.Result.toDeployment(), nil
	}

	msgs, err := json.Marshal(response.DeployPackage.Messages)
	if err != nil {
		return deployment, fmt.Errorf("failed to deploy package and couldn't marshal error messages: %w", err)
	}

	// TODO: better formatting of errors - custom mutation Error type
	return deployment, fmt.Errorf("failed to deploy package: %v", string(msgs))
}

func (d *deployPackageDeployPackageDeploymentPayloadResultDeployment) toDeployment() Deployment {
	return Deployment{
		ID: d.Id,
	}
}
