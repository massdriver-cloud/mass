package terraform

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/spf13/afero"
)

type writeTarget struct {
	label     string
	schema    map[string]interface{}
	writePath string
}

func GenerateFiles(buildPath string, b *bundle.Bundle, fs afero.Fs) error {
	stepsOrDefault := b.Steps

	if stepsOrDefault == nil {
		stepsOrDefault = []bundle.Step{
			{Path: "src", Provisioner: "terraform"},
		}
	}

	for _, step := range stepsOrDefault {
		switch step.Provisioner {
		case "terraform":
			_ = generateFilesForStep(buildPath, step.Path, b, fs)
		}
	}
	return nil
}

func generateFilesForStep(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
	mdVars := map[string]interface{}{
		"required": []string{},
		"properties": map[string]interface{}{
			"md_metadata": map[string]interface{}{
				"type": "object",
			},
		},
	}

	varFileTasks := []writeTarget{
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
		schemaRequiredProperties, _ := createRequiredPropertiesMap(task.schema, task.label)
		content, _ := compile(task.schema["properties"].(map[string]interface{}), schemaRequiredProperties)
		filePath := fmt.Sprintf("/_%s_variables.tf.json", task.label)
		afero.WriteFile(fs, path.Join(buildPath, stepPath, filePath), content, 0755)
	}

	/*
		devParamPath := path.Join(bundlePath, "src", common.DevParamsFilename)
		devParamsVariablesFile, err := os.OpenFile(devParamPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil { // fall back to create missing file
			devParamsVariablesFile, err = os.Create(devParamPath)
			if err != nil {
				return err
			}
		}

		err = CompileDevParams(devParamPath, devParamsVariablesFile)
		if err != nil {
			return fmt.Errorf("error compiling dev params: %w", err)
		}
	*/

	return nil
}

// Compile a JSON Schema to Terraform Variable Definition JSON
func compile(properties map[string]interface{}, requiredProperties map[string]bool) ([]byte, error) {
	// You can't have an empty variable block, so if there are no vars return an empty json block
	if len(properties) == 0 {
		return []byte("{}"), nil
	}

	variableFile := TFVariableFile{
		Variable: makeTFVariablesFromProperties(properties, requiredProperties),
	}

	marshaledVarFile, err := json.MarshalIndent(variableFile, "", "    ")

	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s\n", marshaledVarFile)), nil
}

func makeTFVariablesFromProperties(properties map[string]interface{}, requiredProperties map[string]bool) map[string]TFVariable {
	variables := map[string]TFVariable{}

	for name, prop := range properties {
		variables[name] = newTFVariable(prop.(map[string]interface{}), isRequired(requiredProperties, name))
	}

	return variables
}

/*
Before we were looping through required properties to determine if a proptery is
required for every single property. This map with a tiny value allows us to figure out
if a property is required in an efficient manner.
*/
func createRequiredPropertiesMap(properties map[string]interface{}, label string) (map[string]bool, error) {
	requiredPropertiesMap := make(map[string]bool)
	requiredArray, ok := properties["required"].([]string)

	if !ok {
		return nil, fmt.Errorf("required %s schema properties is not a list", label)
	}

	for _, property := range requiredArray {
		requiredPropertiesMap[property] = true
	}

	return requiredPropertiesMap, nil
}

/*
The two value return for map access will provide the value if found, and a boolean value which
is false if the key is not in the map
*/
func isRequired(requiredProperties map[string]bool, name string) bool {
	_, isRequired := requiredProperties[name]
	return isRequired
}
