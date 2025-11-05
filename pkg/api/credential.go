// Manages credential-type artifacts
package api

import (
	"context"
	"log/slog"
	"os"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// List supported credential types
func ListCredentialTypes(ctx context.Context, mdClient *client.Client) []*ArtifactDefinition {
	response, err := listCredentialArtifactDefinitions(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	if err != nil {
		slog.Error("Failed to fetch credential artifact definitions", "error", err)
		os.Exit(1)
	}

	artifactDefinitions := make([]*ArtifactDefinition, len(response.ArtifactDefinitions))
	for i, def := range response.ArtifactDefinitions {
		artifactDefinitions[i] = &ArtifactDefinition{
			Name: def.Name,
		}
	}

	return artifactDefinitions
}

// Get the first page of credentials for an artifact type
func ListCredentials(ctx context.Context, mdClient *client.Client, artifactType string) ([]*Artifact, error) {
	artifactList := []*Artifact{}
	response, err := getArtifactsByType(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactType)

	for _, artifactRecord := range response.Artifacts.Items {
		artifactList = append(artifactList, artifactRecord.toArtifact(artifactType))
	}

	return artifactList, err
}

// Convert the API response to an Artifact
func (a *getArtifactsByTypeArtifactsPaginatedArtifactsItemsArtifact) toArtifact(artifactType string) *Artifact {
	return &Artifact{
		ID:        a.Id,
		Name:      a.Name,
		Type:      artifactType,
		UpdatedAt: a.UpdatedAt,
	}
}
