package scalars

import (
	"encoding/json"
)

// MarshalJSON marshals a value twice to create an escaped string of JSON
func MarshalJSON(v interface{}) ([]byte, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(bytes))
}

func UnmarshalJSON(data []byte, v *map[string]interface{}) error {
	return json.Unmarshal(data, v)
}
