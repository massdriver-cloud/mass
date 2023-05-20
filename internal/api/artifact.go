package api

type Artifact struct {
	Name string
	ID   string
}

type ArtifactDefinition struct {
	Name string
}

type ArtifactDefinitionWithSchema struct {
	Name   string
	Schema map[string]interface{}
}
