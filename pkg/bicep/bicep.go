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

const paramsFile = "parameters.json"

type InputHandler struct {
	ParamsFile      string
	ConnectionsFile string
}

type InputParameterWrapper struct {
	Value map[string]any `json:"value"`
}

type InputParameters struct {
	Params      InputParameterWrapper `json:"params"`
	MdMetadata  InputParameterWrapper `json:"md_metadata"`
	Connections InputParameterWrapper `json:"connections"`
}

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle, fs afero.Fs) error {
	p := path.Join(buildPath, stepPath, paramsFile)
	params, _ := inputvars.TranspileDevParams(p, b, fs)
	connections, _ := inputvars.TranspileConnectionVarFile(p, b, fs)

	template := InputParameters{
		Params:      InputParameterWrapper{Value: params},
		MdMetadata:  InputParameterWrapper{Value: inputvars.DefaultMdMetadata(b.Name)},
		Connections: InputParameterWrapper{Value: connections},
	}

	content, err := json.MarshalIndent(template, "", "    ")

	if err != nil {
		return err
	}

	return afero.WriteFile(fs, fmt.Sprintf("%s/%s", stepPath, paramsFile), content, 0755)
}

func NewInputHandler() InputHandler {
	return InputHandler{
		ParamsFile:      paramsFile,
		ConnectionsFile: paramsFile,
	}
}

func (i InputHandler) ReadParams(basePath string) ([]byte, error) {
	paramsPath := path.Join(basePath, i.ParamsFile)
	existingData := &InputParameters{}

	err := files.Read(paramsPath, existingData)

	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(existingData.Params.Value)
}

func (i InputHandler) WriteParams(basePath string, params map[string]any) error {
	paramPath := path.Join(basePath, i.ParamsFile)
	existingData := &InputParameters{}

	err := files.Read(paramPath, existingData)

	if err != nil {
		return err
	}

	existingData.Params.Value = params

	return files.Write(paramPath, existingData)
}

// To implement with application example
func (i InputHandler) WriteSecrets(string, map[string]string) error {
	return nil
}

func (i InputHandler) ReadConnections(basePath string) ([]byte, error) {
	connectionsPath := path.Join(basePath, i.ConnectionsFile)
	existingData := &InputParameters{}

	err := files.Read(connectionsPath, &existingData)

	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(existingData.Connections.Value)
}

func (i InputHandler) WriteConnections(basePath string, connections map[string]any) error {
	connectionsPath := path.Join(basePath, i.ConnectionsFile)
	existingData := &InputParameters{}

	err := files.Read(connectionsPath, &existingData)

	if err != nil {
		return err
	}

	existingData.Connections.Value = connections

	return files.Write(connectionsPath, existingData)
}
