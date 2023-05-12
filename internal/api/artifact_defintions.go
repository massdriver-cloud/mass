package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

func GetArtifactDefinitions(client graphql.Client, orgID string) ([]ArtifactDefinitionWithSchema, error) {
	response, err := getArtifactDefinitions(context.Background(), client, orgID)
	return response.toArtifactDefinitions(), err
}

func (ad *getArtifactDefinitionsResponse) toArtifactDefinitions() []ArtifactDefinitionWithSchema {
	var ads []ArtifactDefinitionWithSchema
	for _, artifactDefinition := range ad.ArtifactDefinitions {
		ads = append(ads, ArtifactDefinitionWithSchema(artifactDefinition))
	}

	return ads
}
