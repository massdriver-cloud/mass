package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type ArtifactDefinition struct {
	Name string
}

type ArtifactDefinitionWithSchema struct {
	ID        string         `json:"$id"`
	Name      string         `json:"name"`
	Label     string         `json:"label,omitempty"`
	URL       string         `json:"url,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	Schema    map[string]any `json:"schema"`
}

func GetArtifactDefinition(ctx context.Context, mdClient *client.Client, name string) (*ArtifactDefinitionWithSchema, error) {
	response, err := getArtifactDefinition(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact definition %s: %w", name, err)
	}
	return response.toArtifactDefinition(), nil
}

func (response *getArtifactDefinitionResponse) toArtifactDefinition() *ArtifactDefinitionWithSchema {
	return &ArtifactDefinitionWithSchema{
		ID:        response.ArtifactDefinition.Id,
		Name:      response.ArtifactDefinition.Name,
		Schema:    response.ArtifactDefinition.Schema,
		Label:     response.ArtifactDefinition.Label,
		UpdatedAt: response.ArtifactDefinition.UpdatedAt,
	}
}

func ListArtifactDefinitions(ctx context.Context, mdClient *client.Client) ([]ArtifactDefinitionWithSchema, error) {
	response, err := listArtifactDefinitions(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact definitions: %w", err)
	}
	return response.toArtifactDefinitions(), nil
}

func (response *listArtifactDefinitionsResponse) toArtifactDefinitions() []ArtifactDefinitionWithSchema {
	var ads []ArtifactDefinitionWithSchema
	for _, artifactDefinition := range response.ArtifactDefinitions {
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

func PublishArtifactDefinition(ctx context.Context, mdClient *client.Client, schema map[string]any) (*ArtifactDefinitionWithSchema, error) {
	response, err := publishArtifactDefinition(ctx, mdClient.GQL, mdClient.Config.OrganizationID, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to publish artifact definition: %w", err)
	}
	if !response.PublishArtifactDefinition.Successful {
		return nil, fmt.Errorf("unable to publish artifact definition: %s", response.PublishArtifactDefinition.GetMessages()[0].Message)
	}
	return response.toArtifactDefinition(), nil
}

func (response *publishArtifactDefinitionResponse) toArtifactDefinition() *ArtifactDefinitionWithSchema {
	return &ArtifactDefinitionWithSchema{
		ID:   response.PublishArtifactDefinition.Result.Id,
		Name: response.PublishArtifactDefinition.Result.Name,
	}
}
