package inputvars

import (
	"encoding/json"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/spf13/afero"
)

func DefaultMdMetadata(bundleName string) map[string]interface{} {
	namePrefix := fmt.Sprintf("local-dev-%s-000", bundleName)

	return map[string]interface{}{
		"name_prefix": namePrefix,
		"default_tags": map[string]interface{}{
			"md-project":  "local",
			"md-target":   "dev",
			"md-manifest": bundleName,
			"md-package":  namePrefix,
		},
		"deployment": map[string]interface{}{
			"id": "local-dev-id",
		},
		"observability": map[string]interface{}{
			"alarm_webhook_url": "https://placeholder.com",
		},
	}
}

func TranspileConnectionVarFile(path string, b *bundle.Bundle, fs afero.Fs) (map[string]interface{}, error) {
	var result = map[string]interface{}{}
	emptyConnections := checkEmptySchema(b.Connections)

	if emptyConnections {
		return result, nil
	}

	existingConnectionsVars, err := getExistingVars(path, fs)

	if err != nil {
		return result, err
	}

	connectionsSchemaProperties, ok := b.Connections["properties"].(map[string]interface{})

	if !ok {
		return result, fmt.Errorf("expected connections schema properties to be an object")
	}

	return setValuesIfNotExists(connectionsSchemaProperties, existingConnectionsVars, nil)
}

func TranspileDevParams(path string, b *bundle.Bundle, fs afero.Fs) (map[string]interface{}, error) {
	var result = map[string]interface{}{}
	emptyParams := checkEmptySchema(b.Params)

	if emptyParams {
		return result, nil
	}

	existingParams, err := getExistingVars(path, fs)

	if err != nil {
		return result, err
	}

	var example map[string]interface{}

	if b.Params["examples"] == nil {
		example = make(map[string]interface{})
	} else {
		example, err = getFirstExample(b.Params["examples"].([]interface{}))

		if err != nil {
			return result, err
		}
	}

	paramsSchemaProperties, ok := b.Params["properties"].(map[string]interface{})

	if !ok {
		return result, fmt.Errorf("expected params schema properties to be an object")
	}

	values, err := setValuesIfNotExists(paramsSchemaProperties, existingParams, example)

	if err != nil {
		return result, err
	}

	return values, nil
}

func checkEmptySchema(schema map[string]interface{}) bool {
	return len(schema) == 0
}

func getFirstExample(examples []interface{}) (map[string]interface{}, error) {
	if len(examples) == 0 {
		return make(map[string]interface{}), nil
	}

	firstExample, ok := examples[0].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("expected examples array to contain a list of objects")
	}
	return firstExample, nil
}

func setValuesIfNotExists(paramsSchemaProperties, existingParams map[string]interface{}, example map[string]interface{}) (map[string]interface{}, error) {
	paramsWithExampleOrExistingValue := make(map[string]interface{})
	for propertyName, property := range paramsSchemaProperties {
		result, err := fillDevParam(propertyName, property, existingParams[propertyName], example[propertyName])

		if err != nil {
			return nil, err
		}

		paramsWithExampleOrExistingValue[propertyName] = result
	}

	return paramsWithExampleOrExistingValue, nil
}

var placeholderValue = "REPLACE ME"

func fillDevParam(name string, prop, existingVal, exampleVal interface{}) (interface{}, error) {
	// the base case is we fall back to a placeholder to indicate to the developer they should replace this value.
	var ret interface{} = placeholderValue
	var ok bool

	schemaProperty, ok := prop.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("param %s was not an object", name)
	}

	// handle nested objects recursively
	if schemaProperty["type"] == "object" {
		nestedSchemaProperties, propertiesCastOk := schemaProperty["properties"].(map[string]interface{})

		if !propertiesCastOk {
			return nil, fmt.Errorf("properties block of param %s was not an object", name)
		}

		nestedExampleValuesMap, exampleCastOk := exampleVal.(map[string]interface{})

		if !exampleCastOk {
			nestedExampleValuesMap = make(map[string]interface{})
		}

		existingValue, _ := existingVal.(map[string]interface{})

		return setValuesIfNotExists(nestedSchemaProperties, existingValue, nestedExampleValuesMap)
	}

	if existingVal != nil {
		return existingVal, nil
	}

	if exampleVal != nil {
		return exampleVal, nil
	}

	if schemaProperty["default"] != nil {
		return schemaProperty["default"], nil
	}

	// fall back to an empty array
	if schemaProperty["type"] == "array" {
		return []interface{}{}, nil
	}

	if schemaProperty["type"] == "number" || schemaProperty["type"] == "integer" {
		if minimum, hasMinimum := schemaProperty["minimum"]; hasMinimum {
			return minimum, nil
		}

		return 0, nil
	}

	return ret, nil
}

func getExistingVars(path string, fs afero.Fs) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	content, err := afero.ReadFile(fs, path)

	if err != nil {
		if len(content) == 0 {
			return result, nil
		}
		return nil, err
	}

	marshalErr := json.Unmarshal(content, &result)

	return result, marshalErr
}
