package terraform

import (
	"fmt"
	"os"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
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

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle) error {
	err := generateTfVarsFiles(buildPath, stepPath, b)

	if err != nil {
		return err
	}

	devParamPath := path.Join(buildPath, stepPath, bundle.ParamsFile)

	err = transpileAndWriteDevParams(devParamPath, b)

	if err != nil {
		return fmt.Errorf("error compiling dev params: %w", err)
	}

	err = transpileConnectionVarFile(path.Join(buildPath, stepPath, bundle.ConnsFile), b)

	if err != nil {
		return err
	}

	return nil
}

func generateTfVarsFiles(buildPath, stepPath string, b *bundle.Bundle) error {
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

		props, ok := task.schema["properties"].(map[string]interface{})
		if !ok {
			// We should not hit this now since we are defaulting properties in the bundle
			// unmarshal so if we do get here, we want to know.
			return fmt.Errorf("%s block is missing 'properties' entry", task.label)
		}

		content, err := transpile(props, schemaRequiredProperties)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("/_%s_variables.tf.json", task.label)
		err = os.WriteFile(path.Join(buildPath, stepPath, filePath), content, 0755)

		if err != nil {
			return err
		}
	}

	return nil
}
