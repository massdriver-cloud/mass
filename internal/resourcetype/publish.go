package resourcetype

import (
	"context"
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/jsonschema"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Publish reads, validates, and publishes a resource type from path to the Massdriver API.
func Publish(ctx context.Context, mdClient *client.Client, path string) (*api.ResourceType, error) {
	rt, readErr := Read(ctx, mdClient, path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read resource type: %w", readErr)
	}

	// validate resource type against JSON Schema meta-schema
	// and resource type schema
	rtSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "resource-type.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct resource type schema URL: %w", err)
	}
	err = validateArtifactDefinition(rt, rtSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate resource type schema: %w", err)
	}
	metaSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	err = validateArtifactDefinition(rt, metaSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate resource type against meta schema: %w", err)
	}

	input := api.PublishResourceTypeInput{
		Schema: rt,
	}

	return api.PublishResourceType(ctx, mdClient, input)
}

func validateArtifactDefinition(artDef map[string]any, schemaURL string) error {
	sch, loadErr := jsonschema.LoadSchemaFromURL(schemaURL)
	if loadErr != nil {
		return loadErr
	}
	return jsonschema.ValidateGo(sch, artDef)
}
