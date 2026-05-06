package scalars

import (
	"encoding/json"
	"reflect"
)

// MarshalJSON double-encodes a value into an escaped JSON string for
// transport over Massdriver's `JSON`/`Map` GraphQL scalars.
//
// Empty/nil maps return a nil byte slice. The wrapping json.RawMessage then
// either gets elided (when the field has `omitempty`) or marshals as the bare
// `null` literal (when it doesn't) — both shapes the server accepts. An empty
// non-nil slice would error in encoding/json with "unexpected end of JSON
// input" on no-omitempty fields, so nil is the safe choice.
func MarshalJSON(v any) ([]byte, error) {
	if isEmpty(v) {
		return nil, nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(bytes))
}

// UnmarshalJSON unmarshals raw JSON bytes into the provided map.
func UnmarshalJSON(data []byte, v *map[string]any) error {
	return json.Unmarshal(data, v)
}

// isEmpty reports whether v should be treated as absent — only nil values and
// nil maps/slices qualify. An explicitly empty map (e.g. `map[string]any{}`)
// is *not* empty: it is a valid `Map` value (`{}`) and required fields like
// `CreateDeploymentInput.params` must serialize it that way.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	switch rv.Kind() { //nolint:exhaustive // only nil-able container kinds need the IsNil check
	case reflect.Map, reflect.Slice:
		return rv.IsNil()
	}
	return false
}
