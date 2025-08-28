package provisioners

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
)

type BicepProvisioner struct{}

func (p *BicepProvisioner) ExportMassdriverInputs(stepPath string, variables map[string]any) error {
	// read existing bicep params for this step
	bicepParamsImport := bicep.BicepToSchema(path.Join(stepPath, "template.bicep"))

	newParams := FindMissingFromAirlock(variables, bicepParamsImport.Schema)
	if len(newParams["properties"].(map[string]any)) == 0 {
		return nil
	}

	schemaBytes, marshallErr := json.Marshal(newParams)
	if marshallErr != nil {
		return marshallErr
	}

	content, transpileErr := bicep.SchemaToBicep(bytes.NewReader(schemaBytes))
	if transpileErr != nil {
		return transpileErr
	}

	bicepFile, openErr := os.OpenFile(path.Join(stepPath, "template.bicep"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if openErr != nil {
		return openErr
	}
	defer bicepFile.Close()

	comment := []byte("\n// Auto-generated param declarations from massdriver.yaml\n")
	content = append(comment, content...)
	_, writeErr := bicepFile.Write(content)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (p *BicepProvisioner) ReadProvisionerInputs(stepPath string) (map[string]any, error) {
	bicepParamsImport := bicep.BicepToSchema(path.Join(stepPath, "template.bicep"))

	schemaBytes, marshallErr := json.Marshal(bicepParamsImport.Schema)
	if marshallErr != nil {
		return nil, marshallErr
	}

	variables := map[string]any{}
	marshalErr := json.Unmarshal(schemaBytes, &variables)
	if marshalErr != nil {
		return nil, marshalErr
	}

	return variables, nil
}

func (p *BicepProvisioner) InitializeStep(stepPath string, sourcePath string) error {
	pathInfo, statErr := os.Stat(sourcePath)
	if statErr != nil {
		return statErr
	}
	if pathInfo.IsDir() {
		return errors.New("path is a directory not a bicep template")
	}

	return copyFile(sourcePath, path.Join(stepPath, "template.bicep"))
}
