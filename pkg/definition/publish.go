package definition

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/artifact-definition-schema.json
//go:embed schemas/meta-schema.json
var definitionFS embed.FS

func Publish(c *restclient.MassdriverClient, in io.Reader) error {
	artdefBytes, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	// validate artifact definition against JSON Schema meta-schema
	// and artifact definition schema
	artdefSchemaBytes, _ := definitionFS.ReadFile("schemas/artifact-definition-schema.json")
	if err != nil {
		return err
	}
	err = validateArtifactDefinition(artdefBytes, artdefSchemaBytes)
	if err != nil {
		return err
	}
	metaSchemaBytes, _ := definitionFS.ReadFile("schemas/meta-schema.json")
	if err != nil {
		return err
	}
	err = validateArtifactDefinition(artdefBytes, metaSchemaBytes)
	if err != nil {
		return err
	}

	req := restclient.NewRequest("PUT", "artifact-definitions", bytes.NewBuffer(artdefBytes))
	ctx := context.Background()
	resp, err := c.Do(&ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		respBodyBytes, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return err2
		}
		fmt.Println(string(respBodyBytes))
		return errors.New("received non-200 response from Massdriver: " + resp.Status)
	}

	return nil
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
		return fmt.Errorf(errors)
	}
	return nil
}
