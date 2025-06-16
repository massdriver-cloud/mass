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
	ID        string
	Name      string
	Label     string
	URL       string
	UpdatedAt time.Time
	Schema    map[string]interface{}
}
