package bundle

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/airlock/pkg/terraform"
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
	if b.Params != nil && b.Params.Properties != nil {
		if b.Connections != nil && b.Connections.Properties != nil {
			for param := range b.Params.Properties {
				for connection := range b.Connections.Properties {
					if param == connection {
						return fmt.Errorf("a parameter and connection have the same name: %s", param)
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

	paramsBytes, err := json.Marshal(b.Params)
	if err != nil {
		return nil, err
	}
	paramsSchema, err := schema2json.Parse(bytes.NewReader(paramsBytes))
	if err != nil {
		return nil, err
	}
	connectionsBytes, err := json.Marshal(b.Connections)
	if err != nil {
		return nil, err
	}
	connectionsSchema, err := schema2json.Parse(bytes.NewReader(connectionsBytes))
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

//nolint:gocognit
func matchRequired(input *Schema) error {
	var properties map[string]*Schema

	if input.Properties != nil {
		properties = input.Properties
	} else {
		properties = map[string]*Schema{}
	}

	for _, prop := range properties {
		var propType string
		if prop.Type == "" {
			propType = "object"
		} else {
			propType = prop.Type
		}
		if propType == "object" {
			err := matchRequired(prop)
			if err != nil {
				return err
			}
		}
	}

	if input.Required != nil {
		for _, req := range input.Required {
			if _, propReqOk := properties[req]; !propReqOk {
				return fmt.Errorf("required parameter %s is not defined in properties", req)
			}
		}
	}

	return nil
}

func (b *Bundle) LintParamsMatchVariables() error {
	for _, step := range b.Steps {
		if step.Provisioner == "terraform" || step.Provisioner == "opentofu" {
			paramsConns := map[string]*Schema{}
			if b.Params != nil && b.Params.Properties != nil {
				maps.Copy(paramsConns, b.Params.Properties)
			}
			if b.Connections != nil && b.Connections.Properties != nil {
				maps.Copy(paramsConns, b.Connections.Properties)
			}
			paramsConns["md_metadata"] = &Schema{}

			tfvars, err := terraform.TfToSchema(step.Path)
			if err != nil {
				return err
			}

			missingTfvars := []string{}
			for paramName := range paramsConns {
				match := false
				for prop := tfvars.Properties.Oldest(); prop != nil; prop = prop.Next() {
					if paramName == prop.Key {
						match = true
						break
					}
				}
				if !match && paramName != "md_metadata" {
					missingTfvars = append(missingTfvars, paramName)
				}
			}

			missingParamsConns := []string{}
			for prop := tfvars.Properties.Oldest(); prop != nil; prop = prop.Next() {
				match := false
				for paramName := range paramsConns {
					if paramName == prop.Key {
						match = true
						break
					}
				}
				if !match {
					missingParamsConns = append(missingParamsConns, prop.Key)
				}
			}

			if len(missingParamsConns) > 0 || len(missingTfvars) > 0 {
				err := fmt.Sprintf("missing params or variables detected in step %s:\n", step.Path)

				for _, p := range missingParamsConns {
					err += fmt.Sprintf("\t- variable \"%s\" missing param declaration\n", p)
				}
				for _, v := range missingTfvars {
					err += fmt.Sprintf("\t- param \"%s\" missing variable declaration\n", v)
				}

				return errors.New(err)
			}
		}
	}

	return nil
}
