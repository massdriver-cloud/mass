package terraform

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// transpile a JSON Schema to Terraform Variable Definition JSON
func transpile(properties map[string]interface{}, requiredProperties map[string]bool) ([]byte, error) {
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
		property, ok := prop.(map[string]interface{})
		if !ok {
			// If we hit this something bad happened but it saves the panic and lets us continue
			slog.Warn("Property failed conversion", "name", name, "property", prop)
			continue
		}
		// Validate the property, if there are no keys or a type then skip it or we fail later
		if len(property) == 0 || property["type"] == nil {
			slog.Warn("Property does not have a valid type", "name", name, "property", prop)
			continue
		}
		variables[name] = newTFVariable(property, isRequired(requiredProperties, name))
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
