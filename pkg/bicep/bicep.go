package bicep

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/mass/pkg/inputvars"
	"github.com/spf13/afero"
)

const ParamsFile = "template.parameters.json"

type InputHandler struct {
	ParamsFile      string
	ConnectionsFile string
}

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
	p := path.Join(buildPath, stepPath, ParamsFile)
	params, _ := inputvars.TranspileDevParams(p, b, fs)
	connections, _ := inputvars.TranspileConnectionVarFile(p, b, fs)

	template["parameters"].(map[string]interface{})["params"] = map[string]any{"value": params}
	template["parameters"].(map[string]interface{})["md_metadata"] = map[string]any{"value": inputvars.DefaultMdMetadata(b.Name)}
	template["parameters"].(map[string]interface{})["connections"] = map[string]any{"value": connections}

	content, err := json.MarshalIndent(template, "", "    ")

	if err != nil {
		return err
	}

	return afero.WriteFile(fs, fmt.Sprintf("%s/%s", stepPath, ParamsFile), content, 0755)
}

// reconcileParams reads the params file keeping the md_metadata field intact as it's
// not represented in the UI yet, adds the incoming params, and writes the file back out.
func ReconcileParams(baseDir string, params map[string]any) error {
	paramPath := path.Join(baseDir, ParamsFile)

	fileParams := make(map[string]any)
	err := files.Read(paramPath, &fileParams)
	if err != nil {
		return err
	}

	fileParams["parameters"].(map[string]interface{})["params"] = params

	return files.Write(paramPath, fileParams)
}

func NewInputHandler() InputHandler {
	return InputHandler{
		ParamsFile:      ParamsFile,
		ConnectionsFile: ParamsFile,
	}
}

func (i InputHandler) ReadParams(basePath string) ([]byte, error) {
	paramsPath := path.Join(basePath, i.ParamsFile)
	fileParams := make(map[string]any)

	err := files.Read(paramsPath, &fileParams)

	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(fileParams["parameters"].(map[string]any)["params"].(map[string]any)["value"])
}

func (i InputHandler) WriteParams(basePath string, params map[string]any) error {
	paramPath := path.Join(basePath, i.ParamsFile)

	fileParams := make(map[string]any)
	err := files.Read(paramPath, &fileParams)
	if err != nil {
		return err
	}

	wrappedParams := map[string]any{
		"value": params,
	}

	fileParams["parameters"].(map[string]interface{})["params"] = wrappedParams

	return files.Write(paramPath, fileParams)
}

// To implement with application example
func (i InputHandler) WriteSecrets(basePath string, secrets map[string]string) error {
	return nil
}

func (i InputHandler) ReadConnections(basePath string) ([]byte, error) {
	connectionsPath := path.Join(basePath, i.ConnectionsFile)
	content := make(map[string]any)

	err := files.Read(connectionsPath, &content)

	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(content["parameters"].(map[string]interface{})["connections"].(map[string]any)["value"])
}

func (i InputHandler) WriteConnections(basePath string, connections map[string]any) error {
	connectionsPath := path.Join(basePath, i.ConnectionsFile)

	content := make(map[string]any)
	err := files.Read(connectionsPath, &content)
	if err != nil {
		return err
	}

	wrappedConnections := map[string]any{
		"value": content,
	}

	content["parameters"].(map[string]interface{})["connections"] = wrappedConnections

	return files.Write(connectionsPath, content)
}
