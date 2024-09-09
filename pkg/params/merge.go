package params

import (
	"maps"
)

// Merges two schemas by combining properties and required
func MergeSchemas(m1, m2 map[string]any) map[string]any {
	resultProperties := map[string]any{}
	resultRequired := []any{}

	if m1PropertiesInterface, m1PropertiesExists := m1["properties"]; m1PropertiesExists {
		if m1PropertiesMap, propertiesOk := m1PropertiesInterface.(map[string]any); propertiesOk {
			maps.Copy(resultProperties, m1PropertiesMap)
		}
	}
	if m1RequiredInterface, m1RequiredExists := m1["required"]; m1RequiredExists {
		if m1RequiredSlice, requiredOk := m1RequiredInterface.([]any); requiredOk {
			resultRequired = append(resultRequired, m1RequiredSlice...)
		}
	}

	if m2PropertiesInterface, m2PropertiesExists := m2["properties"]; m2PropertiesExists {
		if m2PropertiesMap, propertiesOk := m2PropertiesInterface.(map[string]any); propertiesOk {
			maps.Copy(resultProperties, m2PropertiesMap)
		}
	}
	if m2RequiredInterface, m2RequiredExists := m2["required"]; m2RequiredExists {
		if m2RequiredSlice, requiredOk := m2RequiredInterface.([]any); requiredOk {
			resultRequired = append(resultRequired, m2RequiredSlice...)
		}
	}

	resultRequired = deduplicateSliceInterface(resultRequired)

	return map[string]any{
		"properties": resultProperties,
		"required":   resultRequired,
	}
}

func deduplicateSliceInterface(slice []any) []any {
	// using a map to perform deduplication
	dedupMap := map[string]bool{}
	result := []any{}

	for _, elem := range slice {
		elemString := elem.(string)
		if _, exists := dedupMap[elemString]; !exists {
			dedupMap[elemString] = true
			result = append(result, elemString)
		}
	}

	return result
}
