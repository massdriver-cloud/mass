package bicep

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/inputvars"
	"github.com/spf13/afero"
)

const paramsFile = "template.parameters.json"

var template = map[string]interface{}{
	"$schema":        "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
	"contentVersion": "1.0.0.0",
	"parameters": map[string]interface{}{
		"params":      map[string]interface{}{},
		"connections": map[string]interface{}{},
		"md_metadata": map[string]interface{}{},
	},
}

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
	p := path.Join(buildPath, stepPath, paramsFile)
	params, _ := inputvars.TranspileDevParams(p, b, fs)
	connections, _ := inputvars.TranspileConnectionVarFile(p, b, fs)

	template["parameters"].(map[string]interface{})["params"] = params
	template["parameters"].(map[string]interface{})["md_metadata"] = inputvars.DefaultMdMetadata(b.Name)
	template["parameters"].(map[string]interface{})["connections"] = connections

	content, err := json.MarshalIndent(template, "", "    ")

	if err != nil {
		return err
	}

	return afero.WriteFile(fs, fmt.Sprintf("%s/%s", stepPath, paramsFile), content, 0755)
}
