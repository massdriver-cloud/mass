package provisioners

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/helm"
)

type HelmProvisioner struct{}

func (p *HelmProvisioner) ExportMassdriverInputs(_ string, _ map[string]any) error {
	// Nothing to do here. Helm doesn't require variables to be declared before use, nor does it require types to be specified
	return nil
}

func (p *HelmProvisioner) ReadProvisionerInputs(stepPath string) (map[string]any, error) {
	helmParamsSchema, err := helm.HelmToSchema(path.Join(stepPath, "values.yaml"))
	if err != nil {
		return nil, err
	}

	schemaBytes, marshallErr := json.Marshal(helmParamsSchema)
	if marshallErr != nil {
		return nil, marshallErr
	}

	variables := map[string]any{}
	err = json.Unmarshal(schemaBytes, &variables)
	if err != nil {
		return nil, err
	}

	return variables, nil
}

func (p *HelmProvisioner) InitializeStep(stepPath string, sourcePath string) error {
	pathInfo, statErr := os.Stat(sourcePath)
	if statErr != nil {
		return statErr
	}
	if !pathInfo.IsDir() {
		return errors.New("path is not a directory containing a helm chart")
	}

	if _, chartErr := os.Stat(path.Join(sourcePath, "Chart.yaml")); errors.Is(chartErr, os.ErrNotExist) {
		return errors.New("path does not contain 'Chart.yaml' file, and therefore isn't a valid Helm chart")
	}
	if _, valuesErr := os.Stat(path.Join(sourcePath, "values.yaml")); errors.Is(valuesErr, os.ErrNotExist) {
		return errors.New("path does not contain 'values.yaml' file, and therefore isn't a valid Helm chart")
	}

	return os.CopyFS(stepPath, os.DirFS(sourcePath))
}
