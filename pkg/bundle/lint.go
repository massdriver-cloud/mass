package bundle

import (
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func (b *Bundle) LintSchema(mdClient *client.Client) LintResult {
	var result LintResult

	bundleSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "bundle.json")
	if err != nil {
		result.AddError("schema-validation", fmt.Sprintf("failed to construct bundle schema URL: %v", err))
		return result
	}

	sch, err := jsonschema.LoadSchemaFromURL(bundleSchemaURL)
	if err != nil {
		result.AddError("schema-validation", fmt.Sprintf("failed to compile bundle schema: %v", err))
		return result
	}

	err = jsonschema.ValidateGo(sch, b)
	if err != nil {
		result.AddError("schema-validation", err.Error())
		return result
	}

	return result
}

func (b *Bundle) LintParamsConnectionsNameCollision() LintResult {
	var result LintResult

	if b.Params != nil {
		if params, ok := b.Params["properties"]; ok {
			if b.Connections != nil {
				if connections, connectionsOk := b.Connections["properties"]; connectionsOk {
					for param := range params.(map[string]any) {
						for connection := range connections.(map[string]any) {
							if param == connection {
								result.AddError("name-collision", fmt.Sprintf("a parameter and connection have the same name: %s", param))
							}
						}
					}
				}
			}
		}
	}
	return result
}

func (b *Bundle) LintMatchRequired() LintResult {
	var result LintResult

	err := matchRequired(b.Params)
	if err != nil {
		result.AddError("required-match", err.Error())
	}

	return result
}

//nolint:gocognit
func matchRequired(input map[string]any) error {
	var properties map[string]any

	if val, propOk := input["properties"]; propOk {
		if properties, propOk = val.(map[string]any); !propOk {
			return fmt.Errorf("properties is not a map[string]any")
		}
	}

	for _, prop := range properties {
		var propType string

		propMap, mapOk := prop.(map[string]any)
		if !mapOk {
			return fmt.Errorf("property is not a map[string]any")
		}

		if val, typeOk := propMap["type"]; typeOk {
			if propType, typeOk = val.(string); !typeOk {
				return fmt.Errorf("type is not a string")
			}
		} else {
			propType = "object"
		}
		if propType == "object" {
			if _, objectOk := propMap["properties"]; objectOk {
				err := matchRequired(propMap)
				if err != nil {
					return err
				}
			}
		}
	}

	var required []string

	if val, reqOk := input["required"]; reqOk {
		requiredInterface, reqIntOk := val.([]any)
		if !reqIntOk {
			return fmt.Errorf("required is not a []any")
		}

		required = make([]string, len(requiredInterface))
		for i, req := range requiredInterface {
			if required[i], reqOk = req.(string); !reqOk {
				return fmt.Errorf("required is not a []string")
			}
		}
	}

	for _, req := range required {
		if _, propReqOk := properties[req]; !propReqOk {
			return fmt.Errorf("required parameter %s is not defined in properties", req)
		}
	}

	return nil
}

//nolint:gocognit
func (b *Bundle) LintInputsMatchProvisioner() LintResult {
	var result LintResult

	massdriverInputs := b.CombineParamsConnsMetadata()
	massdriverInputsProperties, ok := massdriverInputs["properties"].(map[string]any)
	if !ok {
		result.AddError("param-mismatch", "enabled to convert to map[string]interface")
		return result
	}

	for _, step := range b.Steps {
		prov := provisioners.NewProvisioner(step.Provisioner)
		provisionerInputs, err := prov.ReadProvisionerInputs(step.Path)
		if err != nil {
			result.AddError("param-mismatch", err.Error())
			continue
		}
		// If this provisioner doesn't have "ReadProvisionerVariables" implemented, it returns nil
		if provisionerInputs == nil {
			continue
		}
		var provisionerInputsProperties map[string]any
		var exists bool
		if provisionerInputsProperties, exists = provisionerInputs["properties"].(map[string]any); !exists {
			provisionerInputsProperties = map[string]any{}
		}

		missingProvisionerInputs := []string{}
		for name := range massdriverInputsProperties {
			if _, exists = provisionerInputsProperties[name]; !exists {
				missingProvisionerInputs = append(missingProvisionerInputs, name)
			}
		}

		missingMassdriverInputs := []string{}
		for name := range provisionerInputsProperties {
			if _, exists = massdriverInputsProperties[name]; !exists {
				missingMassdriverInputs = append(missingMassdriverInputs, name)
			}
		}

		if len(missingMassdriverInputs) > 0 || len(missingProvisionerInputs) > 0 {
			errMsg := fmt.Sprintf("missing inputs detected in step %s:\n", step.Path)

			for _, p := range missingMassdriverInputs {
				errMsg += fmt.Sprintf("\t- input \"%s\" declared in IaC but missing massdriver.yaml declaration\n", p)
			}
			for _, v := range missingProvisionerInputs {
				errMsg += fmt.Sprintf("\t- input \"%s\" declared in massdriver.yaml but missing IaC declaration\n", v)
			}

			result.AddWarning("param-mismatch", errMsg)
		}
	}

	return result
}
