package commands

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func GetAllArtifacts(client graphql.Client, orgID string, name string) ([]api.Artifact, error) {
	return api.GetAllArtifacts(client, orgID)
}

func GetArtifact(client graphql.Client, orgID string, artifactID string) (*api.Artifact, error) {
	return api.GetArtifact(client, orgID, artifactID)
}
