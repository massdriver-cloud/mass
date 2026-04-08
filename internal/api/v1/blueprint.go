package api

// Blueprint represents the modeled infrastructure for an environment.
type Blueprint struct {
	Instances []Instance `json:"instances,omitempty"`
}
