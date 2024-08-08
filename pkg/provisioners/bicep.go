package provisioners

import (
	"bytes"
	"encoding/json"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
)

type BicepProvisioner struct{}

func (p *BicepProvisioner) ExportMassdriverInputs(stepPath string, variables map[string]interface{}) error {
	// read existing bicep params for this step
	bicepParamsSchema, err := bicep.BicepToSchema(path.Join(stepPath, "template.bicep"))
	if err != nil {
		return err
	}

	newParams := FindMissingFromAirlock(variables, bicepParamsSchema)
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

func (p *BicepProvisioner) ReadProvisionerInputs(stepPath string) (map[string]interface{}, error) {
	bicepParamsSchema, err := bicep.BicepToSchema(path.Join(stepPath, "template.bicep"))
	if err != nil {
		return nil, err
	}

	schemaBytes, marshallErr := json.Marshal(bicepParamsSchema)
	if marshallErr != nil {
		return nil, marshallErr
	}

	variables := map[string]interface{}{}
	err = json.Unmarshal(schemaBytes, &variables)
	if err != nil {
		return nil, err
	}

	return variables, nil
}
