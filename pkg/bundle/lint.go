package bundle

import (
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

func (b *Bundle) LintProvisioners() error {
	return nil
}

func (b *Bundle) LintParamsMatchVariables() error {
	for _, step := range b.Steps {
		if step.Provisioner == "terraform" || step.Provisioner == "opentofu" {
			paramsConns := map[string]interface{}{}
			if params, ok := b.Params["properties"]; ok {
				maps.Copy(paramsConns, params.(map[string]interface{}))
			}
			if conns, ok := b.Connections["properties"]; ok {
				maps.Copy(paramsConns, conns.(map[string]interface{}))
			}
			paramsConns["md_metadata"] = map[string]interface{}{}

			tfvarsString, err := terraform.TfToSchema(step.Path)
			if err != nil {
				return err
			}

			tfvars := map[string]interface{}{}
			err = json.Unmarshal([]byte(tfvarsString), &tfvars)
			if err != nil {
				return err
			}

			missingTfvars := []string{}
			for paramName := range paramsConns {
				match := false
				for tfvarName := range tfvars["properties"].(map[string]interface{}) {
					if paramName == tfvarName {
						match = true
						break
					}
				}
				if !match && paramName != "md_metadata" {
					missingTfvars = append(missingTfvars, paramName)
				}
			}

			missingParamsConns := []string{}
			for tfvarName := range tfvars["properties"].(map[string]interface{}) {
				match := false
				for paramName := range paramsConns {
					if paramName == tfvarName {
						match = true
						break
					}
				}
				if !match {
					missingParamsConns = append(missingParamsConns, tfvarName)
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
