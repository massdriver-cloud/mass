package definition

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/artifact-definition-schema.json
//go:embed schemas/meta-schema.json
var definitionFS embed.FS

func Publish(ctx context.Context, mdClient *client.Client, in io.Reader) error {
	artdefBytes, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	// validate artifact definition against JSON Schema meta-schema
	// and artifact definition schema
	artdefSchemaBytes, err := definitionFS.ReadFile("schemas/artifact-definition-schema.json")
	if err != nil {
		return err
	}
	err = validateArtifactDefinition(artdefBytes, artdefSchemaBytes)
	if err != nil {
		return err
	}
	metaSchemaBytes, err := definitionFS.ReadFile("schemas/meta-schema.json")
	if err != nil {
		return err
	}
	err = validateArtifactDefinition(artdefBytes, metaSchemaBytes)
	if err != nil {
		return err
	}

	var artdefMap map[string]any
	err = json.Unmarshal(artdefBytes, &artdefMap)
	if err != nil {
		return err
	}

	_, err = api.PublishArtifactDefinition(ctx, mdClient, artdefMap)

	return err
}

func validateArtifactDefinition(artdefBytes, schemaBytes []byte) error {
	documentLoader := gojsonschema.NewBytesLoader(artdefBytes)
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		errors := "Artifact definition has schema violations:\n"
		for _, violation := range result.Errors() {
			errors += fmt.Sprintf("\t- %v\n", violation)
		}
		return fmt.Errorf("%s", errors)
	}
	return nil
}
