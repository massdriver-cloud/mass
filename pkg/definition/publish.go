package definition

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func Publish(ctx context.Context, mdClient *client.Client, in io.Reader) error {
	artdefBytes, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("failed to read artifact definition: %w", err)
	}

	// validate artifact definition against JSON Schema meta-schema
	// and artifact definition schema
	artdefSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "artifact-definition.json")
	if err != nil {
		return fmt.Errorf("failed to construct artifact definition schema URL: %w", err)
	}
	err = validateArtifactDefinition(artdefBytes, artdefSchemaURL)
	if err != nil {
		return fmt.Errorf("failed to validate artifact definition schema: %w", err)
	}
	metaSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	err = validateArtifactDefinition(artdefBytes, metaSchemaURL)
	if err != nil {
		return fmt.Errorf("failed to validate artifact definition against meta schema: %w", err)
	}

	var artdefMap map[string]any
	err = json.Unmarshal(artdefBytes, &artdefMap)
	if err != nil {
		return fmt.Errorf("failed to unmarshal artifact definition: %w", err)
	}

	_, err = api.PublishArtifactDefinition(ctx, mdClient, artdefMap)

	return err
}

func validateArtifactDefinition(artdefBytes []byte, schemaURL string) error {
	sch, loadErr := jsonschema.LoadSchemaFromURL(schemaURL)
	if loadErr != nil {
		return loadErr
	}
	return jsonschema.ValidateBytes(sch, artdefBytes)
}
