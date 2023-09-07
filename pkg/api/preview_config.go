package api

type PreviewConfig struct {
	ProjectSlug string                    `json:"projectSlug"`
	Credentials []Credential              `json:"credentials"`
	Packages    map[string]PreviewPackage `json:"packages"`
}

type PreviewPackage struct {
	Params           map[string]interface{} `json:"params"`
	Secrets          []Secret               `json:"secrets,omitempty"`
	RemoteReferences []RemoteRef            `json:"remoteReferences,omitempty"`
}

type RemoteRef struct {
	ArtifactID string `json:"artifactId"`
	Field      string `json:"field"`
}
type Secret struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
}

func (p *PreviewConfig) GetCredentials() []Credential {
	return p.Credentials
}
