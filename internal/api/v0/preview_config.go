package api

// PreviewConfig holds the configuration for deploying a preview environment.
type PreviewConfig struct {
	ProjectSlug     string                    `json:"projectSlug"`
	BaseEnvironment string                    `json:"baseEnvironment,omitempty"`
	Credentials     []Credential              `json:"credentials"`
	Packages        map[string]PreviewPackage `json:"packages"`
}

// PreviewPackage holds per-package configuration used when deploying a preview environment.
type PreviewPackage struct {
	Version          string         `json:"version,omitempty"`
	ReleaseStrategy  string         `json:"releaseStrategy,omitempty"`
	Params           map[string]any `json:"params,omitempty"`
	Secrets          []Secret       `json:"secrets,omitempty"`
	RemoteReferences []RemoteRef    `json:"remoteReferences,omitempty"`
}

// RemoteRef identifies a remote artifact field to use as a connection reference.
type RemoteRef struct {
	ArtifactID string `json:"artifactId"`
	Field      string `json:"field"`
}

// Secret holds a name/value pair for a sensitive configuration value.
type Secret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetCredentials returns the credentials associated with this preview configuration.
func (p *PreviewConfig) GetCredentials() []Credential {
	return p.Credentials
}
