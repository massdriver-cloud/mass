package provisioners

import (
	"bytes"
	"encoding/json"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/terraform"
)

type OpentofuProvisioner struct{}

func (p *OpentofuProvisioner) ExportMassdriverVariables(stepPath string, variables map[string]interface{}) error {
	// read existing OpenTofu variables for this step
	existingTfvarsSchema, err := terraform.TfToSchema(stepPath)
	if err != nil {
		return err
	}

	newVariables := FindMissingFromAirlock(variables, existingTfvarsSchema)
	if len(newVariables["properties"].(map[string]any)) == 0 {
		return nil
	}

	schemaBytes, marshallErr := json.Marshal(newVariables)
	if marshallErr != nil {
		return marshallErr
	}

	content, transpileErr := terraform.SchemaToTf(bytes.NewReader(schemaBytes))
	if transpileErr != nil {
		return transpileErr
	}

	comment := []byte("// Auto-generated variable declarations from massdriver.yaml\n")
	content = append(comment, content...)
	filename := "/_massdriver_variables.tf"
	fh, openErr := os.OpenFile(path.Join(stepPath, filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if openErr != nil {
		return openErr
	}
	defer fh.Close()

	_, writeErr := fh.Write(content)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (p *OpentofuProvisioner) ReadProvisionerVariables(stepPath string) (map[string]interface{}, error) {
	opentofuVariablesSchema, err := terraform.TfToSchema(stepPath)
	if err != nil {
		return nil, err
	}

	schemaBytes, marshallErr := json.Marshal(opentofuVariablesSchema)
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
