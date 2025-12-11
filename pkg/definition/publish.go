package definition

import (
	"context"
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func Publish(ctx context.Context, mdClient *client.Client, path string) (*api.ArtifactDefinitionWithSchema, error) {
	artDef, readErr := Read(ctx, mdClient, path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read artifact definition: %w", readErr)
	}

	// validate artifact definition against JSON Schema meta-schema
	// and artifact definition schema
	artdefSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "artifact-definition.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct artifact definition schema URL: %w", err)
	}
	err = validateArtifactDefinition(artDef, artdefSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate artifact definition schema: %w", err)
	}
	metaSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	err = validateArtifactDefinition(artDef, metaSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate artifact definition against meta schema: %w", err)
	}

	return api.PublishArtifactDefinition(ctx, mdClient, artDef)
}

func validateArtifactDefinition(artDef map[string]any, schemaURL string) error {
	sch, loadErr := jsonschema.LoadSchemaFromURL(schemaURL)
	if loadErr != nil {
		return loadErr
	}
	return jsonschema.ValidateGo(sch, artDef)
}
