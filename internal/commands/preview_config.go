package commands

import "github.com/massdriver-cloud/mass/internal/api"

type PreviewConfig struct {
	Credentials   map[string]string      `json:"credentials"`
	PackageParams map[string]interface{} `json:"packageParams"`
}

func (p *PreviewConfig) GetCredentials() []api.Credential {
	credentials := []api.Credential{}
	for k, v := range p.Credentials {
		cred := api.Credential{
			ArtifactDefinitionType: k,
			ArtifactId:             v,
		}
		credentials = append(credentials, cred)
	}
	return credentials
}
