package resourcetype

import (
	"context"
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/jsonschema"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// Publish reads, validates, and publishes a resource type from path to the Massdriver API.
func Publish(ctx context.Context, mdClient *massdriver.Client, path string) (*ResourceType, error) {
	rt, readErr := Read(ctx, mdClient, path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read resource type: %w", readErr)
	}

	// validate resource type against JSON Schema meta-schema
	// and resource type schema
	cfg := mdClient.Config()
	rtSchemaURL, err := url.JoinPath(cfg.URL, "json-schemas", "resource-type.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct resource type schema URL: %w", err)
	}
	err = validateResourceType(rt, rtSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate resource type schema: %w", err)
	}
	metaSchemaURL, err := url.JoinPath(cfg.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return nil, fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	err = validateResourceType(rt, metaSchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate resource type against meta schema: %w", err)
	}

	return api.PublishResourceType(ctx, mdClient, api.PublishResourceTypeInput{Schema: rt})
}

func validateResourceType(rt map[string]any, schemaURL string) error {
	sch, loadErr := jsonschema.LoadSchemaFromURL(schemaURL)
	if loadErr != nil {
		return loadErr
	}
	return jsonschema.ValidateGo(sch, rt)
}
