package scalars

import (
	"encoding/json"
)

// MarshalJSON marshals a value twice to create an escaped string of JSON
func MarshalJSON(v any) ([]byte, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(bytes))
}

func UnmarshalJSON(data []byte, v *map[string]any) error {
	// GraphQL JSON scalars can be returned as either:
	// 1. A JSON string: "{\"foo\":\"bar\"}"
	// 2. A JSON object: {"foo": "bar"}
	// Try unmarshaling as a string first
	var jsonStr string
	if err := json.Unmarshal(data, &jsonStr); err == nil {
		// Successfully unmarshaled as string, now unmarshal the string's contents
		return json.Unmarshal([]byte(jsonStr), v)
	}
	// Not a string, try unmarshaling directly as JSON object
	return json.Unmarshal(data, v)
}
