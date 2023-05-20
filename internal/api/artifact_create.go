package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

func CreateArtifact(client graphql.Client, orgID string, artifactName string, artifactType string, artifactData map[string]interface{}, artifactSpecs map[string]interface{}) (*Artifact, error) {
	response, err := createArtifact(context.Background(), client, orgID, artifactName, artifactSpecs, artifactType, artifactData)
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
