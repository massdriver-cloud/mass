package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Deployment represents a Massdriver deployment operation and its current status.
type Deployment struct {
	ID                 string    `json:"id"`
	Status             string    `json:"status"`
	Action             string    `json:"action"`
	Version            string    `json:"version"`
	Message            string    `json:"message"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	LastTransitionedAt time.Time `json:"lastTransitionedAt"`
	ElapsedTime        int       `json:"elapsedTime"`
	DeployedBy         string    `json:"deployedBy"`
}

// GetDeployment retrieves a deployment by ID from the Massdriver API.
func GetDeployment(ctx context.Context, mdClient *client.Client, id string) (*Deployment, error) {
	response, err := getDeploymentById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %w", id, err)
	}

	return response.Deployment.toDeployment(), nil
}

func (d *getDeploymentByIdDeployment) toDeployment() *Deployment {
	return &Deployment{
		ID:                 d.Id,
		Status:             string(d.Status),
		Action:             string(d.Action),
		Version:            d.Version,
		Message:            d.Message,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
		LastTransitionedAt: d.LastTransitionedAt,
		ElapsedTime:        d.ElapsedTime,
		DeployedBy:         d.DeployedBy,
	}
}

// ListDeployments returns deployments, optionally filtered.
func ListDeployments(ctx context.Context, mdClient *client.Client, filter *DeploymentsFilter) ([]Deployment, error) {
	response, err := getDeployments(ctx, mdClient.GQL, mdClient.Config.OrganizationID, filter, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]Deployment, 0, len(response.Deployments.Items))
	for _, d := range response.Deployments.Items {
		deployments = append(deployments, Deployment{
			ID:                 d.Id,
			Status:             string(d.Status),
			Action:             string(d.Action),
			Version:            d.Version,
			Message:            d.Message,
			CreatedAt:          d.CreatedAt,
			UpdatedAt:          d.UpdatedAt,
			LastTransitionedAt: d.LastTransitionedAt,
			ElapsedTime:        d.ElapsedTime,
			DeployedBy:         d.DeployedBy,
		})
	}

	return deployments, nil
}
