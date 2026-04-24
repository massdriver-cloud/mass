package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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
	CreatedAt          time.Time `json:"createdAt,omitzero" mapstructure:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt,omitzero" mapstructure:"updatedAt"`
	LastTransitionedAt time.Time `json:"lastTransitionedAt,omitzero" mapstructure:"lastTransitionedAt"`
	Instance           *Instance `json:"instance,omitempty" mapstructure:"instance,omitempty"`
}

// DeploymentLog is a single batch of logs emitted by the provisioner during a deployment.
// The message may span multiple lines separated by "\n".
type DeploymentLog struct {
	Timestamp time.Time `json:"timestamp,omitzero" mapstructure:"timestamp"`
	Message   string    `json:"message" mapstructure:"message"`
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

// GetDeploymentLogs returns all log batches emitted for the given deployment so far, oldest first.
func GetDeploymentLogs(ctx context.Context, mdClient *client.Client, deploymentID string) ([]DeploymentLog, error) {
	response, err := getDeploymentLogs(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for deployment %s: %w", deploymentID, err)
	}

	logs := make([]DeploymentLog, 0, len(response.Deployment.Logs))
	for _, l := range response.Deployment.Logs {
		log := DeploymentLog{}
		if decodeErr := decode(l, &log); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode deployment log: %w", decodeErr)
		}
		logs = append(logs, log)
	}
	return logs, nil
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
	if err := decode(v, &dep); err != nil {
		return nil, fmt.Errorf("failed to decode deployment: %w", err)
	}
	return &dep, nil
}
