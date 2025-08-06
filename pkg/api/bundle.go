package api

type Bundle struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Spec        map[string]any `json:"spec,omitempty"`
	SpecVersion string         `json:"specVersion,omitempty"`
}
