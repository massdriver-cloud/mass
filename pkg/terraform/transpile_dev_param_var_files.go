package terraform

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

func transpileAndWriteDevParams(path string, b *bundle.Bundle) error {
	emptyParams := checkEmptySchema(b.Params)

	if emptyParams {
		err := os.WriteFile(path, []byte("{}"), 0755)

		if err != nil {
			return err
		}

		return nil
	}

	existingParams, err := getExistingVars(path)

	if err != nil {
		return err
	}

	var example map[string]interface{}

	if b.Params["examples"] == nil {
		example = make(map[string]interface{})
	} else {
		example, err = getFirstExample(b.Params["examples"].([]interface{}))

		if err != nil {
			return err
		}
	}

	paramsSchemaProperties, ok := b.Params["properties"].(map[string]interface{})

	if !ok {
		return fmt.Errorf("expected params schema properties to be an object")
	}

	result, err := setValuesIfNotExists(paramsSchemaProperties, existingParams, example)

	if err != nil {
		fmt.Println(err)
		return err
	}

	resultWithMdMetadata := mergeMdMetadata(result, b.Name)

	bytes, err := json.MarshalIndent(resultWithMdMetadata, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path, bytes, 0755)

	return err
}

func mergeMdMetadata(params map[string]interface{}, bundleName string) map[string]interface{} {
	namePrefix := fmt.Sprintf("local-dev-%s-000", bundleName)
	defaultMetadata := map[string]interface{}{
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

	// if md_metadata is not set, initialize it to a reasonable starting point
	if _, ok := params["md_metadata"]; !ok {
		params["md_metadata"] = defaultMetadata
	} else {
		// merge md metadata ties go to existing values
		for k, v := range defaultMetadata {
			if _, ok2 := params["md_metadata"].(map[string]interface{})[k]; !ok2 {
				params["md_metadata"].(map[string]interface{})[k] = v
			}
		}
	}

	return params
}

func getExistingVars(path string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	content, err := os.ReadFile(path)

	if err != nil {
		if len(content) == 0 {
			return result, nil
		}
		return nil, err
	}

	marshalErr := json.Unmarshal(content, &result)

	return result, marshalErr
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
