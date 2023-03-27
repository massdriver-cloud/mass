package commands

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/schema2json"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/bundle-schema.json
var bundleFS embed.FS

func Lint(b *bundle.Bundle) error {
	err := LintSchema(b)
	if err != nil {
		return err
	}

	err = LintParamsConnectionsNameCollision(b)
	if err != nil {
		return err
	}

	return nil
}

func LintSchema(b *bundle.Bundle) error {
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

func LintParamsConnectionsNameCollision(b *bundle.Bundle) error {
	if b.Params != nil {
		if params, ok := b.Params["properties"]; ok {
			if b.Connections != nil {
				if connections, ok := b.Connections["properties"]; ok {
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

func LintEnvs(b *bundle.Bundle) (map[string]string, error) {
	result := map[string]string{}

	input, err := buildEnvsInput(b)
	if err != nil {
		return nil, fmt.Errorf("error building env query: %s", err.Error())
	}

	for name, query := range b.AppSpec.Envs {
		jq, err := gojq.Parse(query)
		if err != nil {
			return result, errors.New("The jq query for environment variable " + name + " is invalid: " + err.Error())
		}

		iter := jq.Run(input)
		value, ok := iter.Next()
		if !ok || value == nil {
			return result, errors.New("The jq query for environment variable " + name + " didn't produce a result")
		}
		if err, ok := value.(error); ok {
			return result, errors.New("The jq query for environment variable " + name + " produced an error: " + err.Error())
		}
		var valueString string
		if valueString, ok = value.(string); !ok {
			resultBytes, err := json.Marshal(value)
			if err != nil {
				return result, errors.New("The jq query for environment variable " + name + " produced an uninterpretable value: " + err.Error())
			}
			valueString = string(resultBytes)
		}
		_, multiple := iter.Next()
		if multiple {
			return result, errors.New("The jq query for environment variable " + name + " produced multiple values, which isn't supported")
		}
		result[name] = valueString
	}

	return result, nil
}

func buildEnvsInput(b *bundle.Bundle) (map[string]interface{}, error) {
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
