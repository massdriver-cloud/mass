package bicep

import (
	"bytes"
	"encoding/json"
	"os"
	"path"

	"github.com/massdriver-cloud/airlock/pkg/bicep"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func GenerateFiles(buildPath, stepPath string, b *bundle.Bundle) error {
	err := generateBicepParams(buildPath, stepPath, b)
	if err != nil {
		return err
	}

	return nil
}

func generateBicepParams(buildPath, stepPath string, b *bundle.Bundle) error {

	// read existing bicep params for this step
	bicepParamsSchema, err := bicep.BicepToSchema(path.Join(buildPath, stepPath, "template.bicep"))
	if err != nil {
		return err
	}

	mdYamlVariables := provisioners.CombineParamsConnsMetadata(b)

	newParams := provisioners.FindMissingFromAirlock(mdYamlVariables, bicepParamsSchema)
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

	bicepFile, openErr := os.OpenFile(path.Join(buildPath, stepPath, "template.bicep"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
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
