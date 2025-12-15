package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Deployment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type DeploymentLog struct {
	Content   string
	Step      string
	Timestamp time.Time
	Index     int
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

func GetDeploymentLogs(ctx context.Context, mdClient *client.Client, deploymentID string) ([]DeploymentLog, error) {
	response, err := getDeploymentLogStream(ctx, mdClient.GQL, mdClient.Config.OrganizationID, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}

	logs := make([]DeploymentLog, len(response.DeploymentLogStream.Logs))
	for i, log := range response.DeploymentLogStream.Logs {
		logs[i] = DeploymentLog{
			Content:   log.Content,
			Step:      log.Metadata.Step,
			Timestamp: log.Metadata.Timestamp,
			Index:     log.Metadata.Index,
		}
	}

	return logs, nil
}
