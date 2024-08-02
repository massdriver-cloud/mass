package terraform

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/massdriver-cloud/airlock/pkg/terraform"
	"github.com/massdriver-cloud/mass/pkg/bundle"
)

//go:embed schemas/metadata-schema.json
var bundleFS embed.FS

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle) error {
	err := generateTfVarsFiles(buildPath, stepPath, b)
	if err != nil {
		return err
	}

	return nil
}

func generateTfVarsFiles(buildPath, stepPath string, b *bundle.Bundle) error {
	metadataBytes, err := bundleFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return err
	}

	var metadata map[string]interface{}
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return err
	}

	// read existing terraform variables for this step
	existingTfvarsSchema, err := terraform.TfToSchema(path.Join(buildPath, stepPath))
	if err != nil {
		return err
	}
	existingTfvarsNames := []string{}
	for tfvar := existingTfvarsSchema.Properties.Oldest(); tfvar != nil; tfvar = tfvar.Next() {
		existingTfvarsNames = append(existingTfvarsNames, tfvar.Key)
	}

	type task struct {
		label     string
		schema    *bundle.Schema
		writePath string
	}
	varFileTasks := []task{
		{
			label:     "params",
			schema:    b.Params,
			writePath: path.Join(buildPath, stepPath),
		},
		{
			label:     "connections",
			schema:    b.Connections,
			writePath: path.Join(buildPath, stepPath),
		},
		// {
		// 	label:     "md",
		// 	schema:    metadata,
		// 	writePath: path.Join(buildPath, stepPath),
		// },
	}

	for _, task := range varFileTasks {
		newVariables := &bundle.Schema{
			Required:   []string{},
			Properties: map[string]*bundle.Schema{},
		}

		// if _, exists := task.schema["properties"]; exists {
		// 	taskProperties = task.schema["properties"].(map[string]any)
		// }
		// if _, exists := task.schema["required"]; exists {
		// 	taskRequired = task.schema["required"].([]any)
		// }

		// check each variable in the schema, and if doesn't already exist as a declared variable in the terraform, add it to be rendered
		for key, value := range task.schema.Properties {
			if !slices.Contains(existingTfvarsNames, key) {
				newVariables.Properties[key] = value
				for _, elem := range task.schema.Required {
					if key == elem {
						newVariables.Required = append(newVariables.Required, key)
					}

				}
			}
		}

		if len(newVariables.Properties) == 0 {
			break
		}

		schemaBytes, marshallErr := json.Marshal(newVariables)
		if marshallErr != nil {
			return marshallErr
		}

		content, transpileErr := terraform.SchemaToTf(bytes.NewReader(schemaBytes))
		if transpileErr != nil {
			return transpileErr
		}

		filePath := fmt.Sprintf("/_%s_variables.tf", task.label)
		fh, openErr := os.OpenFile(path.Join(buildPath, stepPath, filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if openErr != nil {
			return openErr
		}
		defer fh.Close()

		_, writeErr := fh.Write(content)
		if writeErr != nil {
			return writeErr
		}
	}

	return nil
}
