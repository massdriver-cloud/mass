package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// ArtifactDefinition represents a minimal artifact definition with just a name.
type ArtifactDefinition struct {
	Name string
}

// ArtifactDefinitionWithSchema represents an artifact definition including its full JSON schema.
type ArtifactDefinitionWithSchema struct {
	ID        string         `json:"$id" mapstructure:"$id"`
	Name      string         `json:"name" mapstructure:"name"`
	Label     string         `json:"label,omitempty" mapstructure:"label"`
	URL       string         `json:"url,omitempty" mapstructure:"url"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
	Schema    map[string]any `json:"schema" mapstructure:"schema"`
}

// GetArtifactDefinition fetches a single artifact definition by name from the API.
func GetArtifactDefinition(ctx context.Context, mdClient *client.Client, name string) (*ArtifactDefinitionWithSchema, error) {
	split := strings.Split(name, "/")
	if len(split) != 2 {
		name = strings.Join([]string{mdClient.Config.OrganizationID, name}, "/")
	}
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

// ListArtifactDefinitions returns all artifact definitions for the configured organization.
func ListArtifactDefinitions(ctx context.Context, mdClient *client.Client) ([]ArtifactDefinitionWithSchema, error) {
	response, err := listArtifactDefinitions(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact definitions: %w", err)
	}
	return response.toArtifactDefinitions(), nil
}

func (response *listArtifactDefinitionsResponse) toArtifactDefinitions() []ArtifactDefinitionWithSchema {
	definitions := make([]ArtifactDefinitionWithSchema, len(response.ArtifactDefinitions))
	for i, def := range response.ArtifactDefinitions {
		definitions[i] = ArtifactDefinitionWithSchema{
			ID:        def.Id,
			Name:      def.Name,
			Schema:    def.Schema,
			Label:     def.Label,
			UpdatedAt: def.UpdatedAt,
		}
	}
	return definitions
}

// DeleteArtifactDefinition deletes an artifact definition by name from the Massdriver API.
func DeleteArtifactDefinition(ctx context.Context, mdClient *client.Client, name string) (*ArtifactDefinitionWithSchema, error) {
	split := strings.Split(name, "/")
	if len(split) != 2 {
		name = strings.Join([]string{mdClient.Config.OrganizationID, name}, "/")
	}
	response, err := deleteArtifactDefinition(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete artifact definition %s: %w", name, err)
	}
	if !response.DeleteArtifactDefinition.Successful {
		messages := response.DeleteArtifactDefinition.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to delete artifact definition:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to delete artifact definition")
	}
	// Check if result is empty (genqlient generates value types, not pointers)
	if response.DeleteArtifactDefinition.Result.Id == "" {
		return nil, errors.New("delete artifact definition returned no result")
	}
	return &ArtifactDefinitionWithSchema{
		ID:   response.DeleteArtifactDefinition.Result.Id,
		Name: response.DeleteArtifactDefinition.Result.Name,
	}, nil
}

// PublishArtifactDefinition publishes a new or updated artifact definition to the Massdriver API.
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
