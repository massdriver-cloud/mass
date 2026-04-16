package api

// ResourceType is a type of resource that an instance can produce or consume. Examples include "database", "cache", and "queue".
type ResourceType struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}
