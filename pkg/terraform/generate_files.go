package terraform

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/terraform"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/spf13/afero"
)

//go:embed schemas/metadata-schema.json
var bundleFS embed.FS

type schema struct {
	label     string
	schema    map[string]interface{}
	writePath string
}

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
	err := generateTfVarsFiles(buildPath, stepPath, b, fs)

	if err != nil {
		return err
	}

	devParamPath := path.Join(buildPath, stepPath, bundle.ParamsFile)

	err = transpileAndWriteDevParams(devParamPath, b, fs)

	if err != nil {
		return fmt.Errorf("error compiling dev params: %w", err)
	}

	err = transpileConnectionVarFile(path.Join(buildPath, stepPath, bundle.ConnsFile), b, fs)

	if err != nil {
		return err
	}

	return nil
}

func generateTfVarsFiles(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
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
	existingTfvarsString, err := terraform.TfToSchema(path.Join(buildPath, stepPath))
	if err != nil {
		return err
	}
	existingTfvars := map[string]interface{}{}
	err = json.Unmarshal([]byte(existingTfvarsString), &existingTfvars)
	if err != nil {
		return err
	}

	varFileTasks := []schema{
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
		{
			label:     "md",
			schema:    metadata,
			writePath: path.Join(buildPath, stepPath),
		},
	}

	for _, task := range varFileTasks {
		newVariables := map[string]interface{}{
			"required":   []string{},
			"properties": map[string]any{},
		}

		// check each variable in the schema, and if doesn't already exist as a declared variable in the terraform, add it to be rendered
		for key, value := range task.schema["properties"].(map[string]any) {
			if _, exists := existingTfvars["properties"]; exists {
				existingTfvarsProperties := existingTfvars["properties"].(map[string]any)

				if _, exists := existingTfvarsProperties[key]; !exists {
					newVariables["properties"].(map[string]any)[key] = value
					if _, exists := existingTfvars["required"]; exists {
						existingTfvarsRequired := existingTfvars["required"].([]interface{})

						for _, elem := range existingTfvarsRequired {
							if key == elem.(string) {
								newVariables["required"] = append(newVariables["required"].([]string), key)
							}
						}
					}
				}
			}
		}

		if len(newVariables["properties"].(map[string]any)) == 0 {
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
		fh, openErr := fs.OpenFile(path.Join(buildPath, stepPath, filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
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
