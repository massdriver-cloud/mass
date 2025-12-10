package provisioners

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/opentofu"
)

type OpentofuProvisioner struct{}

func (p *OpentofuProvisioner) ExportMassdriverInputs(stepPath string, variables map[string]any) error {
	// read existing OpenTofu variables for this step
	tofuVarsImport := opentofu.TofuToSchema(stepPath)
	if tofuVarsImport.Schema == nil {
		return errors.New("failed to read existing OpenTofu variable declarations: " + tofuVarsImport.PrettyDiags())
	}

	newVariables := FindMissingFromAirlock(variables, tofuVarsImport.Schema)
	if len(newVariables["properties"].(map[string]any)) == 0 {
		return nil
	}

	schemaBytes, marshallErr := json.Marshal(newVariables)
	if marshallErr != nil {
		return marshallErr
	}

	content, transpileErr := opentofu.SchemaToTofu(bytes.NewReader(schemaBytes))
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

func (p *OpentofuProvisioner) ReadProvisionerInputs(stepPath string) (map[string]any, error) {
	tofuVarsImport := opentofu.TofuToSchema(stepPath)

	schemaBytes, marshallErr := json.Marshal(tofuVarsImport.Schema)
	if marshallErr != nil {
		return nil, marshallErr
	}

	variables := map[string]any{}
	unmarshalErr := json.Unmarshal(schemaBytes, &variables)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return variables, nil
}

func (p *OpentofuProvisioner) InitializeStep(stepPath string, sourcePath string) error {
	pathInfo, statErr := os.Stat(sourcePath)
	if statErr != nil {
		return statErr
	}
	if !pathInfo.IsDir() {
		return errors.New("path is not a directory, cannot initialize")
	}

	// remove the dummy main.tf if we are copying from a source
	maintfPath := path.Join(stepPath, "main.tf")
	if _, maintfErr := os.Stat(maintfPath); maintfErr == nil {
		err := os.Remove(maintfPath)
		if err != nil {
			return err
		}
	}

	// intentionally not ignoring the .terraform.lock.hcl file since it should be copied
	ignorePatterns := []string{
		".terraform",
		"*.tfstate",
		"*.tfstate.backup",
		"*.tfvars",
		"*.tfvars.json",
	}
	return copyDir(sourcePath, stepPath, ignorePatterns)
}
