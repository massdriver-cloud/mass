package terraform

import (
	"os"
	"path"

	"github.com/massdriver-cloud/mass/pkg/files"
)

type InputHandler struct {
	paramsFile      string
	connectionsFile string
	secretsFile     string
}

func NewInputHandler() InputHandler {
	return InputHandler{
		paramsFile:      ParamsFile,
		connectionsFile: ConnsFile,
		secretsFile:     SecretsFile,
	}
}

func (i InputHandler) ReadParams(basePath string) ([]byte, error) {
	paramsPath := path.Join(basePath, i.paramsFile)
	return os.ReadFile(paramsPath)
}

func (i InputHandler) WriteParams(basePath string, params map[string]any) error {
	paramPath := path.Join(basePath, ParamsFile)

	fileParams := make(map[string]any)
	err := files.Read(paramPath, &fileParams)
	if err != nil {
		return err
	}

	combinedParams := make(map[string]any)
	if v, ok := fileParams["md_metadata"]; ok {
		combinedParams["md_metadata"] = v
	}

	for k, v := range params {
		combinedParams[k] = v
	}

	return files.Write(paramPath, combinedParams)
}

func (i InputHandler) WriteSecrets(basePath string, secrets map[string]string) error {
	secretsPath := path.Join(basePath, i.secretsFile)
	return files.Write(secretsPath, secrets)
}

func (i InputHandler) ReadConnections(basePath string) ([]byte, error) {
	connsPath := path.Join(basePath, i.connectionsFile)
	return os.ReadFile(connsPath)
}

func (i InputHandler) WriteConnections(basePath string, connections map[string]any) error {
	connsPath := path.Join(basePath, i.connectionsFile)
	return files.Write(connsPath, connections)
}
