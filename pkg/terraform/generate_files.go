package terraform

import (
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/spf13/afero"
)

type schema struct {
	label     string
	schema    map[string]interface{}
	writePath string
}

var mdVars = map[string]interface{}{
	"required": []string{},
	"properties": map[string]interface{}{
		"md_metadata": map[string]interface{}{
			"type": "object",
		},
	},
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
			schema:    mdVars,
			writePath: path.Join(buildPath, stepPath),
		},
	}

	for _, task := range varFileTasks {
		schemaRequiredProperties := createRequiredPropertiesMap(task.schema)

		content, err := transpile(task.schema["properties"].(map[string]interface{}), schemaRequiredProperties)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("/_%s_variables.tf.json", task.label)
		err = afero.WriteFile(fs, path.Join(buildPath, stepPath, filePath), content, 0755)

		if err != nil {
			return err
		}
	}

	return nil
}
