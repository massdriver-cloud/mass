package terraform

import (
	"encoding/json"
	"fmt"
)

// Compile a JSON Schema to Terraform Variable Definition JSON
func compile(properties map[string]interface{}, requiredProperties map[string]bool) ([]byte, error) {
	// You can't have an empty variable block, so if there are no vars return an empty json block
	if len(properties) == 0 {
		return []byte("{}"), nil
	}

	variableFile := TFVariableFile{
		Variable: makeTFVariablesFromProperties(properties, requiredProperties),
	}

	marshaledVarFile, err := json.MarshalIndent(variableFile, "", "    ")

	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s\n", marshaledVarFile)), nil
}

func makeTFVariablesFromProperties(properties map[string]interface{}, requiredProperties map[string]bool) map[string]TFVariable {
	variables := map[string]TFVariable{}

	for name, prop := range properties {
		variables[name] = newTFVariable(prop.(map[string]interface{}), isRequired(requiredProperties, name))
	}

	return variables
}

/*
Before we were looping through required properties to determine if a proptery is
required for every single property. This map with a tiny value allows us to figure out
if a property is required in an efficient manner.
*/
func createRequiredPropertiesMap(properties map[string]interface{}) map[string]bool {
	requiredPropertiesMap := make(map[string]bool)

	requiredArray, ok := properties["required"].([]string)

	if !ok {
		return requiredPropertiesMap
	}

	for _, property := range requiredArray {
		requiredPropertiesMap[property] = true
	}

	return requiredPropertiesMap
}

/*
The two value return for map access will provide the value if found, and a boolean value which
is false if the key is not in the map
*/
func isRequired(requiredProperties map[string]bool, name string) bool {
	_, isRequired := requiredProperties[name]
	return isRequired
}
