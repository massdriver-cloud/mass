// Manages credential-type artifacts
package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

var credentialArtifactDefinitions = []ArtifactDefinition{
	{"massdriver/aws-iam-role"},
	{"massdriver/azure-service-principal"},
	{"massdriver/gcp-service-account"},
	{"massdriver/kubernetes-cluster"},
}

// List supported credential types
func ListCredentialTypes() []ArtifactDefinition {
	return credentialArtifactDefinitions
}

// Get the first page of credentials for an artifact type
func ListCredentials(client graphql.Client, orgID string, artifactType string) ([]*Artifact, error) {
	artifactList := []*Artifact{}
	response, err := getArtifactsByType(context.Background(), client, orgID, artifactType)

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
