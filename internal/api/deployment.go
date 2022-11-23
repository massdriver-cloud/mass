package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type Deployment struct {
	ID     string
	Status string
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
