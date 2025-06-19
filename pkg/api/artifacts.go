package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Artifact struct {
	Name string
	ID   string
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
