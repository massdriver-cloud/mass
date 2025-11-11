package api

import (
	"context"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Deployment struct {
	ID                string         `json:"id"`
	Status            string         `json:"status"`
	Action            string         `json:"action,omitempty"`
	Version           string         `json:"version,omitempty"`
	Message           string         `json:"message,omitempty"`
	Params            map[string]any `json:"params,omitempty"`
	DeployedBy        string         `json:"deployedBy,omitempty"`
	CreatedAt         time.Time      `json:"createdAt,omitempty"`
	UpdatedAt         time.Time      `json:"updatedAt,omitempty"`
	LastTransitionedAt *time.Time    `json:"lastTransitionedAt,omitempty"`
	ElapsedTime       int            `json:"elapsedTime,omitempty"`
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

func (d *decommissionPackageDecommissionPackageDeploymentPayloadResultDeployment) toDeployment() *Deployment {
	return &Deployment{
		ID: d.Id,
	}
}
