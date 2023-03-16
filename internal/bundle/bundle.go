package bundle

type Step struct {
	Path        string `json:"path" yaml:"path"`
	Provisioner string `json:"provisioner" yaml:"provisioner"`
}

type Bundle struct {
	Schema      string                 `json:"schema" yaml:"schema"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	SourceURL   string                 `json:"source_url" yaml:"source_url"`
	Type        string                 `json:"type" yaml:"type"`
	Access      string                 `json:"access" yaml:"access"`
	Steps       []Step                 `json:"steps" yaml:"steps"`
	Artifacts   map[string]interface{} `json:"artifacts" yaml:"artifacts"`
	Params      map[string]interface{} `json:"params" yaml:"params"`
	Connections map[string]interface{} `json:"connections" yaml:"connections"`
	UI          map[string]interface{} `json:"ui" yaml:"ui"`
	AppSpec     AppSpec                `json:"app" yaml:"app"`
}

type AppSpec struct {
	Envs     map[string]string `json:"envs" yaml:"envs"`
	Policies []string          `json:"policies" yaml:"policies"`
}

func (b *Bundle) IsInfrastructure() bool {
	// a Deprecation warning is printed in the bundle parse function
	return b.Type == "bundle" || b.Type == "infrastructure"
}

func (b *Bundle) IsApplication() bool {
	return b.Type == "application"
}
