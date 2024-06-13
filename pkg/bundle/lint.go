package bundle

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/schema2json"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/bundle-schema.json
//go:embed schemas/meta-schema.json
var bundleFS embed.FS

func (b *Bundle) LintSchema() error {
	schemaBytes, _ := bundleFS.ReadFile("schemas/bundle-schema.json")
	documentLoader := gojsonschema.NewGoLoader(b)
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		errors := "massdriver.yaml has schema violations:\n"
		for _, violation := range result.Errors() {
			errors += fmt.Sprintf("\t- %v\n", violation)
		}
		return fmt.Errorf(errors)
	}
	return nil
}

func (b *Bundle) LintParamsConnectionsNameCollision() error {
	if b.Params != nil {
		if params, ok := b.Params["properties"]; ok {
			if b.Connections != nil {
				if connections, connectionsOk := b.Connections["properties"]; connectionsOk {
					for param := range params.(map[string]interface{}) {
						for connection := range connections.(map[string]interface{}) {
							if param == connection {
								return fmt.Errorf("a parameter and connection have the same name: %s", param)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

const jqErrorPrefix = "the jq query for environment variable "

func (b *Bundle) LintEnvs() (map[string]string, error) {
	result := map[string]string{}

	if b.AppSpec == nil {
		return result, nil
	}

	input, err := b.buildEnvsInput()
	if err != nil {
		return nil, fmt.Errorf("error building env query: %w", err)
	}

	for name, query := range b.AppSpec.Envs {
		jq, parseErr := gojq.Parse(query)
		if parseErr != nil {
			return result, errors.New(jqErrorPrefix + name + " is invalid: " + parseErr.Error())
		}

		iter := jq.Run(input)
		value, ok := iter.Next()
		if !ok || value == nil {
			return result, errors.New(jqErrorPrefix + name + " didn't produce a result")
		}
		if castErr, castOk := value.(error); castOk {
			return result, errors.New(jqErrorPrefix + name + " produced an error: " + castErr.Error())
		}
		var valueString string
		if valueString, ok = value.(string); !ok {
			resultBytes, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return result, errors.New(jqErrorPrefix + name + " produced an uninterpretable value: " + marshalErr.Error())
			}
			valueString = string(resultBytes)
		}
		_, multiple := iter.Next()
		if multiple {
			return result, errors.New(jqErrorPrefix + name + " produced multiple values, which isn't supported")
		}
		result[name] = valueString
	}

	return result, nil
}

func (b *Bundle) buildEnvsInput() (map[string]interface{}, error) {
	result := map[string]interface{}{}

	paramsSchema, err := schema2json.ParseMapStringInterface(b.Params)
	if err != nil {
		return nil, err
	}
	connectionsSchema, err := schema2json.ParseMapStringInterface(b.Connections)
	if err != nil {
		return nil, err
	}
	result["params"], err = schema2json.GenerateJSON(paramsSchema)
	if err != nil {
		return nil, err
	}
	result["connections"], err = schema2json.GenerateJSON(connectionsSchema)
	if err != nil {
		return nil, err
	}

	secrets := map[string]interface{}{}
	for name := range b.AppSpec.Secrets {
		secrets[name] = "some-secret-value"
	}
	result["secrets"] = secrets

	return result, nil
}

func (b *Bundle) LintMatchRequired() error {
	return matchRequired(b.Params)
}

func matchRequired(input map[string]interface{}) error {
	var properties map[string]interface{}

	if val, ok := input["properties"]; ok {
		properties = val.(map[string]interface{})
	}

	for _, prop := range properties {
		var propType string

		propMap := prop.(map[string]interface{})

		if val, ok := propMap["type"]; ok {
			propType = val.(string)
		} else {
			propType = "object"
		}
		if propType == "object" {
			if val, ok := propMap["properties"]; ok {
				err := matchRequired(val.(map[string]interface{}))
				if err != nil {
					return err
				}
			}
		}
	}

	var required []string

	if val, ok := input["required"]; ok {
		required = val.([]string)
	}

	for _, req := range required {
		if _, ok := properties[req]; !ok {
			return fmt.Errorf("required parameter %s is not defined in properties", req)
		}
	}

	return nil
}
