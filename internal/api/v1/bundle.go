package api

import (
	"time"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID          string    `json:"id" mapstructure:"id"`
	Name        string    `json:"name" mapstructure:"name"`
	Version     string    `json:"version" mapstructure:"version"`
	Description string    `json:"description,omitempty" mapstructure:"description"`
	Icon        string    `json:"icon,omitempty" mapstructure:"icon"`
	SourceURL   string    `json:"sourceUrl,omitempty" mapstructure:"sourceUrl"`
	CreatedAt   time.Time `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}
