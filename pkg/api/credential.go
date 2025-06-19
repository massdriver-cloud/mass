// Manages credential-type artifacts
package api

import (
	"context"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

var credentialArtifactDefinitions = []*ArtifactDefinition{
	{"massdriver/aws-iam-role"},
	{"massdriver/azure-service-principal"},
	{"massdriver/gcp-service-account"},
	{"massdriver/kubernetes-cluster"},
}

// List supported credential types
func ListCredentialTypes() []*ArtifactDefinition {
	return credentialArtifactDefinitions
}

// Get the first page of credentials for an artifact type
func ListCredentials(ctx context.Context, mdClient *client.Client, artifactType string) ([]*Artifact, error) {
	artifactList := []*Artifact{}
	response, err := getArtifactsByType(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactType)

	for _, artifactRecord := range response.Artifacts.Items {
		artifactList = append(artifactList, artifactRecord.toArtifact())
	}

	return artifactList, err
}

// Convert the API response to an Artifact
func (a *getArtifactsByTypeArtifactsPaginatedArtifactsItemsArtifact) toArtifact() *Artifact {
	return &Artifact{
		ID:   a.Id,
		Name: a.Name,
	}
}
