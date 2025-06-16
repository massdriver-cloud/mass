package api

import "time"

type Artifact struct {
	Name string
	ID   string
}

type ArtifactDefinition struct {
	Name string
}

type ArtifactDefinitionWithSchema struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Label     string                 `json:"label"`
	URL       string                 `json:"url"`
	UpdatedAt time.Time             `json:"updatedAt"`
	Schema    map[string]interface{} `json:"schema"`
}
