package api

// Instance represents a deployed bundle instance within a Massdriver environment.
type Instance struct {
	ID              string       `json:"id" mapstructure:"id"`
	Name            string       `json:"name" mapstructure:"name"`
	Status          string       `json:"status" mapstructure:"status"`
	Version         string       `json:"version" mapstructure:"version"`
	ReleaseStrategy string       `json:"releaseStrategy" mapstructure:"releaseStrategy"`
	Environment     *Environment `json:"environment,omitempty" mapstructure:"environment,omitempty"`
	Bundle          *Bundle      `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
}
