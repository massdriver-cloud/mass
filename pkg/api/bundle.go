package api

type Bundle struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Spec    map[string]any `json:"spec,omitempty"`
}
