package terraform

import (
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

var mdVars = map[string]interface{}{
	"required": []string{},
	"properties": map[string]interface{}{
		"md_metadata": map[string]interface{}{
			"type": "object",
		},
	},
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
	err := generateTfVarsFiles(buildPath, stepPath, b, fs)

	if err != nil {
		return err
	}

	devParamPath := path.Join(buildPath, stepPath, "_params.auto.tfvars.json")

	err = compileAndWriteDevParams(devParamPath, b, fs)

	if err != nil {
		return fmt.Errorf("error compiling dev params: %w", err)
	}

	err = compileConnectionVarFile(path.Join(buildPath, stepPath, "_connections.auto.tfvars.json"), b, fs)

	if err != nil {
		return err
	}

	return nil
}

func generateTfVarsFiles(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
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
		schemaRequiredProperties, err := createRequiredPropertiesMap(task.schema, task.label)

		if err != nil {
			return err
		}

		content, err := compile(task.schema["properties"].(map[string]interface{}), schemaRequiredProperties)

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
