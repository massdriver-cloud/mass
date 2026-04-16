package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Deployment represents a record of an infrastructure provisioning operation.
type Deployment struct {
	ID                 string    `json:"id" mapstructure:"id"`
	Status             string    `json:"status" mapstructure:"status"`
	Action             string    `json:"action" mapstructure:"action"`
	Version            string    `json:"version" mapstructure:"version"`
	Message            string    `json:"message,omitempty" mapstructure:"message"`
	DeployedBy         string    `json:"deployedBy,omitempty" mapstructure:"deployedBy"`
	ElapsedTime        int       `json:"elapsedTime" mapstructure:"elapsedTime"`
	CreatedAt          time.Time `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
	LastTransitionedAt time.Time `json:"lastTransitionedAt,omitempty" mapstructure:"lastTransitionedAt"`
	Instance           *Instance `json:"instance,omitempty" mapstructure:"instance,omitempty"`
}

// GetDeployment retrieves a deployment by ID from the Massdriver API.
func GetDeployment(ctx context.Context, mdClient *client.Client, id string) (*Deployment, error) {
	response, err := getDeployment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %w", id, err)
	}

	return toDeployment(response.Deployment)
}

// ListDeployments returns deployments, optionally filtered.
func ListDeployments(ctx context.Context, mdClient *client.Client, filter *DeploymentsFilter) ([]Deployment, error) {
	response, err := listDeployments(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]Deployment, 0, len(response.Deployments.Items))
	for _, resp := range response.Deployments.Items {
		dep, depErr := toDeployment(resp)
		if depErr != nil {
			return nil, fmt.Errorf("failed to convert deployment: %w", depErr)
		}
		deployments = append(deployments, *dep)
	}

	return deployments, nil
}

// CreateDeployment starts a new deployment for an instance.
func CreateDeployment(ctx context.Context, mdClient *client.Client, instanceID string, input CreateDeploymentInput) (*Deployment, error) {
	response, err := createDeployment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, instanceID, input)
	if err != nil {
		return nil, err
	}
	if !response.CreateDeployment.Successful {
		messages := response.CreateDeployment.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to create deployment:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to create deployment")
	}
	return toDeployment(response.CreateDeployment.Result)
}

func toDeployment(v any) (*Deployment, error) {
	dep := Deployment{}
	if err := mapstructure.Decode(v, &dep); err != nil {
		return nil, fmt.Errorf("failed to decode deployment: %w", err)
	}
	return &dep, nil
}
