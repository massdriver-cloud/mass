package provisioners

import (
	"slices"

	"github.com/massdriver-cloud/airlock/pkg/schema"
)

func FindMissingFromAirlock(mdParamsSchema map[string]any, airlockParams *schema.Schema) map[string]any {
	mdProperties := map[string]any{}
	mdRequired := []any{}

	var ok bool
	if _, exists := mdParamsSchema["properties"]; exists {
		mdProperties, ok = mdParamsSchema["properties"].(map[string]any)
		if !ok {
			return nil
		}
	}
	if _, exists := mdParamsSchema["required"]; exists {
		mdRequired, ok = mdParamsSchema["required"].([]any)
		if !ok {
			return nil
		}
	}

	airlockParamsNames := []string{}
	for airlockParam := airlockParams.Properties.Oldest(); airlockParam != nil; airlockParam = airlockParam.Next() {
		airlockParamsNames = append(airlockParamsNames, airlockParam.Key)
	}

	missingProperties := map[string]any{}
	missingRequired := []any{}

	// check each variable in the massdriver schema, and if doesn't already exist as a declared variable in the airlock, add it to the list of missing
	for key, value := range mdProperties {
		if !slices.Contains(airlockParamsNames, key) {
			missingProperties[key] = value
			for _, elem := range mdRequired {
				if key == elem.(string) {
					missingRequired = append(missingRequired, key)
				}
			}
		}
	}

	return map[string]any{
		"properties": missingProperties,
		"required":   missingRequired,
	}
}

func FindMissingFromMassdriver(airlockInputsSchema map[string]any, mdParamsSchema map[string]any) map[string]any {
	mdProperties := map[string]any{}
	var ok bool

	if _, exists := mdParamsSchema["properties"]; exists {
		mdProperties, ok = mdParamsSchema["properties"].(map[string]any)
		if !ok {
			return nil
		}
	}

	airlockProperties := map[string]any{}
	airlockRequired := []any{}

	if _, exists := airlockInputsSchema["properties"]; exists {
		airlockProperties, ok = airlockInputsSchema["properties"].(map[string]any)
		if !ok {
			return nil
		}
	}
	if _, exists := airlockInputsSchema["required"]; exists {
		airlockRequired, ok = airlockInputsSchema["required"].([]any)
		if !ok {
			return nil
		}
	}

	missingProperties := map[string]any{}
	missingRequired := []any{}

	// check each variable in the massdriver schema, and if doesn't already exist as a declared variable in the airlock, add it to the list of missing
	for airlockParamName, airlockParamValue := range airlockProperties {
		if _, exists := mdProperties[airlockParamName]; !exists {
			missingProperties[airlockParamName] = airlockParamValue
			for _, elem := range airlockRequired {
				if airlockParamName == elem.(string) {
					missingRequired = append(missingRequired, airlockParamName)
				}
			}
		}
	}

	return map[string]any{
		"properties": missingProperties,
		"required":   missingRequired,
	}
}
