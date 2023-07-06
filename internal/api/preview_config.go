package api

type PreviewConfig struct {
	ProjectSlug   string                 `json:"projectSlug"`
	Credentials   map[string]string      `json:"credentials"`
	PackageParams map[string]interface{} `json:"packageParams"`
}

func (p *PreviewConfig) GetCredentials() []Credential {
	credentials := []Credential{}
	for k, v := range p.Credentials {
		cred := Credential{
			ArtifactDefinitionType: k,
			ArtifactId:             v,
		}
		credentials = append(credentials, cred)
	}
	return credentials
}
