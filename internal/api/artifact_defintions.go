package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

func GetArtifactDefinitions(client graphql.Client, orgID string, input ArtifactDefinitionInput) ([]ArtifactDefinitionWithSchema, error) {
	response, err := getArtifactDefinitions(context.Background(), client, orgID, input)
	return response.toArtifactDefinitions(), err
}

func (ad *getArtifactDefinitionsResponse) toArtifactDefinitions() []ArtifactDefinitionWithSchema {
	var ads []ArtifactDefinitionWithSchema
	for _, artifactDefinition := range ad.ArtifactDefinitions {
		ads = append(ads, ArtifactDefinitionWithSchema{Name: artifactDefinition.Name, Schema: artifactDefinition.Schema})
	}

	return ads
}
