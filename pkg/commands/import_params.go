package commands

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	yaml3 "gopkg.in/yaml.v3"
)

func ImportParams(buildPath string, skipVerify bool) error {
	fmt.Println("Checking IaC for missing parameters...")

	b := bundle.Bundle{}
	var node yaml3.Node

	mdYamlPath := path.Join(buildPath, "massdriver.yaml")
	fileBytes, readErr := os.ReadFile(mdYamlPath)
	if readErr != nil {
		return readErr
	}

	unmashalNodeErr := yaml3.Unmarshal(fileBytes, &node)
	if unmashalNodeErr != nil {
		return unmashalNodeErr
	}
	unmarshalBundleErr := yaml3.Unmarshal(fileBytes, &b)
	if unmarshalBundleErr != nil {
		return unmarshalBundleErr
	}

	missing := map[string]any{}
	for _, step := range stepsOrDefault(b.Steps) {
		prov := provisioners.NewProvisioner(step.Provisioner)
		inputs, err := prov.ReadProvisionerInputs(path.Join(buildPath, step.Path))
		if err != nil {
			return err
		}
		maps.Copy(missing, provisioners.FindMissingFromMassdriver(inputs, b.CombineParamsConnsMetadata()))
	}

	if !skipVerify {
		var verifyError error
		missing, verifyError = verifyImport(missing)
		if verifyError != nil {
			return verifyError
		}
	}

	if len(missing["properties"].(map[string]any)) == 0 {
		fmt.Println("No missing parameters found.")
		return nil
	}

	var encodedMissing yaml3.Node
	encodedMissing.Encode(missing)

	// Walk the params node to find the properties and required nodes
	var paramsNodeValue *yaml3.Node
	var paramsNodePropertiesNodeValue *yaml3.Node
	var paramsNodeRequiredNodeValue *yaml3.Node
	for ii := 0; ii < len(node.Content[0].Content); ii += 2 {
		iiNodeName := node.Content[0].Content[ii]
		if iiNodeName.Value == "params" {
			paramsNodeValue = node.Content[0].Content[ii+1]
			for jj := 0; jj < len(paramsNodeValue.Content); jj += 2 {
				jjNodeName := paramsNodeValue.Content[jj]
				if jjNodeName.Value == "properties" {
					paramsNodePropertiesNodeValue = paramsNodeValue.Content[jj+1]
					paramsNodePropertiesNodeValue.Style = 0
				}
				if jjNodeName.Value == "required" {
					paramsNodeRequiredNodeValue = paramsNodeValue.Content[jj+1]
				}
			}
			break
		}
	}

	// If params node doesn't contain properties or required, add them
	if paramsNodePropertiesNodeValue == nil {
		paramsNodePropertiesNodeValue = &yaml3.Node{
			Kind:  yaml3.MappingNode,
			Style: yaml3.FlowStyle,
		}
		paramsNodeValue.Content = append(paramsNodeValue.Content, &yaml3.Node{Kind: yaml3.ScalarNode, Value: "properties", Style: 0}, paramsNodePropertiesNodeValue)
	}
	if paramsNodeRequiredNodeValue == nil {
		paramsNodeRequiredNodeValue = &yaml3.Node{
			Kind: yaml3.SequenceNode,
		}
		paramsNodeValue.Content = append(paramsNodeValue.Content, &yaml3.Node{Kind: yaml3.ScalarNode, Value: "required", Style: 0}, paramsNodeRequiredNodeValue)
	}

	var missingPropertiesNodeValue *yaml3.Node
	var missingRequiredNodeValue *yaml3.Node
	for kk := 0; kk < len(encodedMissing.Content); kk += 2 {
		kkNodeName := encodedMissing.Content[kk]
		if kkNodeName.Value == "properties" {
			missingPropertiesNodeValue = encodedMissing.Content[kk+1]
		}
		if kkNodeName.Value == "required" {
			missingRequiredNodeValue = encodedMissing.Content[kk+1]
		}
	}

	paramsNodePropertiesNodeValue.Content = append(paramsNodePropertiesNodeValue.Content, missingPropertiesNodeValue.Content...)
	paramsNodeRequiredNodeValue.Content = append(paramsNodeRequiredNodeValue.Content, missingRequiredNodeValue.Content...)

	newBytes, marshalErr := yaml3.Marshal(&node)
	if marshalErr != nil {
		return marshalErr
	}

	writeErr := os.WriteFile(mdYamlPath, newBytes, 0644)
	if writeErr != nil {
		return writeErr
	}

	fmt.Println("Updated massdriver.yaml with missing parameters.")

	return nil
}

func verifyImport(params map[string]any) (map[string]any, error) {
	importedProperties := map[string]any{}
	paramsToImport := map[string]any{
		"properties": importedProperties,
		"required":   []any{},
	}

	missingProperties := map[string]any{}
	if _, ok := params["properties"]; ok {
		missingProperties = params["properties"].(map[string]any)
	}
	missingRequired := []any{}
	if _, ok := params["required"]; ok {
		missingRequired = params["required"].([]any)
	}

	for paramName := range missingProperties {
		prompt := promptui.Prompt{
			Label:     "Would you like to import the parameter \"" + paramName + "\"",
			Default:   "y",
			IsConfirm: true,
		}

		validate := func(s string) error {
			if len(s) == 1 && strings.Contains("YyNn", s) || prompt.Default != "" && len(s) == 0 {
				return nil
			}
			return fmt.Errorf("\"%s\" is not a valid response, must be \"y\" or \"n\"", s)
		}
		prompt.Validate = validate

		_, err := prompt.Run()
		confirmed := !errors.Is(err, promptui.ErrAbort)

		if confirmed {
			importedProperties[paramName] = missingProperties[paramName]
			for _, req := range missingRequired {
				if req.(string) == paramName {
					paramsToImport["required"] = append(paramsToImport["required"].([]any), paramName)
				}
			}
		}
	}

	return paramsToImport, nil
}
