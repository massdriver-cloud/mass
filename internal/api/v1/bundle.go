package api

import (
	"time"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}
