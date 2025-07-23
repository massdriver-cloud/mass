package bundle

var paramsTransformations = []func(map[string]any) error{EnsureBooleansHaveDefault}

func ApplyTransformations(schema map[string]any, transformations []func(map[string]any) error) error {
	for _, transformation := range transformations {
		err := transformation(schema)
		if err != nil {
			return err
		}
	}

	for _, v := range schema {
		if nestedSchema, isObject := v.(map[string]any); isObject {
			err := ApplyTransformations(nestedSchema, transformations)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// EnsureBooleansHaveDefault ensures that boolean types have a default value set to false if not already defined.
// This is due to an oddity in RJSF where booleans without a default value are treated as undefined, which can violate 'required' constraints.
func EnsureBooleansHaveDefault(schema map[string]any) error {
	if schemaType, ok := schema["type"]; ok && schemaType == "boolean" {
		if _, ok := schema["default"]; !ok {
			schema["default"] = false
		}
	}
	return nil
}
