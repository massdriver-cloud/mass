package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type RemoteReference struct {
	Artifact Artifact `json:"artifact"`
}

type Artifact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Field     string    `json:"field,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

func CreateArtifact(ctx context.Context, mdClient *client.Client, artifactName string, artifactType string, artifactData map[string]any, artifactSpecs map[string]any) (*Artifact, error) {
	response, err := createArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactName, artifactSpecs, artifactType, artifactData)
	if err != nil {
		return nil, err
	}
	if !response.CreateArtifact.Successful {
		return nil, fmt.Errorf("unable to create artifact: %s", response.CreateArtifact.GetMessages()[0].Message)
	}
	return response.CreateArtifact.toArtifact(), err
}

func (payload *createArtifactCreateArtifactArtifactPayload) toArtifact() *Artifact {
	return &Artifact{
		Name: payload.Result.Name,
		ID:   payload.Result.Id,
	}
}

func DownloadArtifact(ctx context.Context, mdClient *client.Client, artifactID string) (string, error) {
	response, err := downloadArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID, DownloadFormatRaw)
	if err != nil {
		return "", fmt.Errorf("failed to download artifact %s: %w", artifactID, err)
	}

	return response.DownloadArtifact.RenderedArtifact, nil
}
