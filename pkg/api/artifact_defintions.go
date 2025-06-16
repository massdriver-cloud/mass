package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

func ListArtifactDefinitions(client graphql.Client, orgID string) ([]ArtifactDefinitionWithSchema, error) {
	response, err := listArtifactDefinitions(context.Background(), client, orgID)
	return response.toArtifactDefinitions(), err
}

func (ad *listArtifactDefinitionsResponse) toArtifactDefinitions() []ArtifactDefinitionWithSchema {
	var ads []ArtifactDefinitionWithSchema
	for _, artifactDefinition := range ad.ArtifactDefinitions {
		ads = append(ads, ArtifactDefinitionWithSchema{
			ID:        artifactDefinition.Id,
			Name:      artifactDefinition.Name,
			Schema:    artifactDefinition.Schema,
			Label:     artifactDefinition.Label,
			UpdatedAt: artifactDefinition.UpdatedAt,
		})
	}

	return ads
}
