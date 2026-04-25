// Package api provides a client for the Massdriver v1 GraphQL API.
package api

// Blueprint represents the modeled infrastructure for an environment.
type Blueprint struct {
	Instances []Instance `json:"instances,omitempty"`
}
