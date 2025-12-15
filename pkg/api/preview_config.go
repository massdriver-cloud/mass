package api

type PreviewConfig struct {
	ProjectSlug     string                    `json:"projectSlug"`
	BaseEnvironment string                    `json:"baseEnvironment,omitempty"`
	Credentials     []Credential              `json:"credentials"`
	Packages        map[string]PreviewPackage `json:"packages"`
}

type PreviewPackage struct {
	Version          string         `json:"version,omitempty"`
	ReleaseStrategy  string         `json:"releaseStrategy,omitempty"`
	Params           map[string]any `json:"params,omitempty"`
	Secrets          []Secret       `json:"secrets,omitempty"`
	RemoteReferences []RemoteRef    `json:"remoteReferences,omitempty"`
}

type RemoteRef struct {
	ArtifactID string `json:"artifactId"`
	Field      string `json:"field"`
}
type Secret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (p *PreviewConfig) GetCredentials() []Credential {
	return p.Credentials
}
