package bundle

import (
	"maps"
)

// CombineParamsConnsMetadata merges the bundle's params, connections, and metadata schemas into one map.
func (b *Bundle) CombineParamsConnsMetadata() map[string]any {
	combined := map[string]any{
		"properties": map[string]any{},
		"required":   []any{},
	}

	for _, sch := range []map[string]any{b.Params, b.Connections, MetadataSchema} {
		if _, exists := sch["properties"]; exists {
			combinedProps, ok1 := combined["properties"].(map[string]any)
			schProps, ok2 := sch["properties"].(map[string]any)
			if ok1 && ok2 {
				maps.Copy(combinedProps, schProps)
			}
		}
		if _, exists := sch["required"]; exists {
			combinedReq, ok1 := combined["required"].([]any)
			schReq, ok2 := sch["required"].([]any)
			if ok1 && ok2 {
				combined["required"] = append(combinedReq, schReq...)
			}
		}
	}

	return combined
}
