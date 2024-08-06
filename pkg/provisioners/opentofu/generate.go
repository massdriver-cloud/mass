package opentofu

import (
	"bytes"
	"encoding/json"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/terraform"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle) error {
	err := generateTfVarsFiles(buildPath, stepPath, b)
	if err != nil {
		return err
	}

	return nil
}

func generateTfVarsFiles(buildPath, stepPath string, b *bundle.Bundle) error {

	// read existing OpenTofu variables for this step
	existingTfvarsSchema, err := terraform.TfToSchema(path.Join(buildPath, stepPath))
	if err != nil {
		return err
	}

	mdYamlVariables := provisioners.CombineParamsConnsMetadata(b)

	newVariables := provisioners.FindMissingFromAirlock(mdYamlVariables, existingTfvarsSchema)
	if len(newVariables["properties"].(map[string]any)) == 0 {
		return nil
	}

	schemaBytes, marshallErr := json.Marshal(newVariables)
	if marshallErr != nil {
		return marshallErr
	}

	content, transpileErr := terraform.SchemaToTf(bytes.NewReader(schemaBytes))
	if transpileErr != nil {
		return transpileErr
	}

	comment := []byte("// Auto-generated variable declarations from massdriver.yaml\n")
	content = append(comment, content...)
	filename := "/_massdriver_variables.tf"
	fh, openErr := os.OpenFile(path.Join(buildPath, stepPath, filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if openErr != nil {
		return openErr
	}
	defer fh.Close()

	_, writeErr := fh.Write(content)
	if writeErr != nil {
		return writeErr
	}

	return nil
}
