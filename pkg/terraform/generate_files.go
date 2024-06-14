package terraform

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
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
		schemaBytes, marshallErr := json.Marshal(task.schema)
		if marshallErr != nil {
			return marshallErr
		}

		content, transpileErr := terraform.SchemaToTf(bytes.NewReader(schemaBytes))
		if transpileErr != nil {
			return transpileErr
		}

		filePath := fmt.Sprintf("/_%s_variables.tf", task.label)
		err = afero.WriteFile(fs, path.Join(buildPath, stepPath, filePath), content, 0755)

		if err != nil {
			return err
		}
	}

	return nil
}
