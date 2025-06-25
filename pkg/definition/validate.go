package definition

import (
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/xeipuuv/gojsonschema"
)

func Validate(mdClient *client.Client, artdefBytes []byte) error {
	artdefSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "artifact-definition.json")
	if err != nil {
		return fmt.Errorf("failed to construct artifact definition schema URL: %w", err)
	}
	if err := validateFromURL(artdefBytes, artdefSchemaURL); err != nil {
		return fmt.Errorf("failed to validate artifact definition against artifact definition schema: %w", err)
	}

	metaschemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	if err := validateFromURL(artdefBytes, metaschemaURL); err != nil {
		return fmt.Errorf("failed to validate artifact definition against meta schema: %w", err)
	}
	return nil
}

func validateFromURL(artdefBytes []byte, schemaURL string) error {
	documentLoader := gojsonschema.NewBytesLoader(artdefBytes)
	schemaLoader := gojsonschema.NewReferenceLoader(schemaURL)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("failed to validate artifact definition: %w", err)
	}

	if !result.Valid() {
		errors := "artifact definition has schema violations:\n"
		for _, violation := range result.Errors() {
			errors += fmt.Sprintf("\t- %v\n", violation)
		}
		return fmt.Errorf("%s", errors)
	}
	return nil
}
