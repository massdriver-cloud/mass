package provisioners

import (
	"slices"

	"github.com/massdriver-cloud/airlock/pkg/schema"
)

func FindMissingFromAirlock(mdParamsSchema map[string]any, airlockParams *schema.Schema) map[string]any {
	mdProperties := map[string]any{}
	mdRequired := []any{}

	if _, exists := mdParamsSchema["properties"]; exists {
		mdProperties = mdParamsSchema["properties"].(map[string]any)
	}
	if _, exists := mdParamsSchema["required"]; exists {
		mdRequired = mdParamsSchema["required"].([]any)
	}

	airlockParamsNames := []string{}
	for tfvar := airlockParams.Properties.Oldest(); tfvar != nil; tfvar = tfvar.Next() {
		airlockParamsNames = append(airlockParamsNames, tfvar.Key)
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
