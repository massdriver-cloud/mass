package api

// Component represents a blueprint component (formerly known as a manifest).
type Component struct {
	ID          string `json:"id" mapstructure:"id"`
	Name        string `json:"name" mapstructure:"name"`
	Description string `json:"description,omitempty" mapstructure:"description"`
}
