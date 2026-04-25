package api

import (
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

// decode copies a genqlient-generated struct into one of our exported API structs.
// We use mapstructure because it correctly handles custom-scalar maps (Map, JSON) that
// our genqlient bindings already decode to map[string]any — JSON roundtripping instead
// would re-invoke MarshalJSON on those and produce the doubly-encoded wire form.
//
// But mapstructure treats time.Time as a generic struct: it tries to flatten it into a
// map (hitting only unexported fields → empty map) and then re-inflate it. Both passes
// lose the value. The decode hook below intercepts each direction so timestamps survive.
func decode(input, output any) error {
	cfg := &mapstructure.DecoderConfig{
		Result:     output,
		DecodeHook: timeWrapHook,
	}
	dec, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return err
	}
	return dec.Decode(input)
}

const timeWrapKey = "__time__"

var (
	timeType    = reflect.TypeOf(time.Time{})
	timePtrType = reflect.TypeOf(&time.Time{})
)

// timeWrapHook preserves time.Time values through mapstructure's internal struct→map→struct
// dance by wrapping them in a single-key map on the way out and unwrapping on the way in.
func timeWrapHook(from, to reflect.Type, data any) (any, error) {
	if from == timeType || from == timePtrType {
		var t time.Time
		switch d := data.(type) {
		case time.Time:
			t = d
		case *time.Time:
			if d != nil {
				t = *d
			}
		default:
			return data, nil
		}
		return map[string]any{timeWrapKey: t.Format(time.RFC3339Nano)}, nil
	}
	if to == timeType {
		if m, ok := data.(map[string]any); ok {
			if s, ok := m[timeWrapKey].(string); ok {
				return time.Parse(time.RFC3339Nano, s)
			}
		}
	}
	return data, nil
}
