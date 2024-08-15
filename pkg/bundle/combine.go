package bundle

import (
	"maps"
)

func (b *Bundle) CombineParamsConnsMetadata() map[string]interface{} {
	combined := map[string]any{
		"properties": map[string]any{},
		"required":   []any{},
	}

	for _, sch := range []map[string]any{b.Params, b.Connections, MetadataSchema} {
		if _, exists := sch["properties"]; exists {
			maps.Copy(combined["properties"].(map[string]any), sch["properties"].(map[string]any))
		}
		if _, exists := sch["required"]; exists {
			combined["required"] = append(combined["required"].([]any), sch["required"].([]any)...)
		}
	}

	return combined
}
