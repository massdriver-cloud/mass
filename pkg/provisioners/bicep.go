// Package provisioners provides implementations for various infrastructure provisioners.
package provisioners

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
)

// BicepProvisioner implements Provisioner for Azure Bicep templates.
type BicepProvisioner struct{}

// ExportMassdriverInputs appends auto-generated Bicep param declarations from the massdriver schema to the step's template.
func (p *BicepProvisioner) ExportMassdriverInputs(stepPath string, variables map[string]any) error {
	// read existing bicep params for this step
	bicepParamsImport := bicep.BicepToSchema(path.Join(stepPath, "template.bicep"))
	if bicepParamsImport.Schema == nil {
		return errors.New("failed to read existing Bicep param declarations: " + bicepParamsImport.PrettyDiags())
	}

	newParams := FindMissingFromAirlock(variables, bicepParamsImport.Schema)
	newParamsProps, newParamsPropsOk := newParams["properties"].(map[string]any)
	if !newParamsPropsOk {
		return errors.New("failed to get properties from missing params")
	}
	if len(newParamsProps) == 0 {
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

	bicepFile, openErr := os.OpenFile(path.Join(stepPath, "template.bicep"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
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

// ReadProvisionerInputs reads the Bicep parameter declarations from the step's template file.
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

// InitializeStep copies the source Bicep template file into the step directory.
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
