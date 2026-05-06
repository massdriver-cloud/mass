package cli

// AttributesToAnyMap converts cobra's StringToString flag value into the
// map[string]any shape the API inputs expect. Returns nil when no entries
// are present so the field marshals to JSON null and the server treats it
// as absent rather than rejecting an empty Map.
func AttributesToAnyMap(attrs map[string]string) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

// StringMapToAnyMap preserves an existing attribute map (read off a
// Project, Environment, or Component) when an update command needs to
// round-trip it without modification.
func StringMapToAnyMap(m map[string]string) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
