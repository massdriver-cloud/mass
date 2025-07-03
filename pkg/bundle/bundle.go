package bundle

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
)

//go:embed schemas/metadata-schema.json
var embedFS embed.FS

var MetadataSchema = parseMetadataSchema()

const (
	ParamsFile = "_params.auto.tfvars.json"
	ConnsFile  = "_connections.auto.tfvars.json"
)

type Step struct {
	Path         string         `json:"path,omitempty" yaml:"path,omitempty"`
	Provisioner  string         `json:"provisioner,omitempty" yaml:"provisioner,omitempty"`
	SkipOnDelete bool           `json:"skip_on_delete,omitempty" yaml:"skip_on_delete,omitempty"`
	Config       map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
}

type Bundle struct {
	Schema      string         `json:"schema,omitempty" yaml:"schema,omitempty"`
	Name        string         `json:"name,omitempty" yaml:"name,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	SourceURL   string         `json:"source_url,omitempty" yaml:"source_url,omitempty"`
	Type        string         `json:"type,omitempty" yaml:"type,omitempty"`
	Access      string         `json:"access,omitempty" yaml:"access,omitempty"`
	Steps       []Step         `json:"steps,omitempty" yaml:"steps,omitempty"`
	Artifacts   map[string]any `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Params      map[string]any `json:"params,omitempty" yaml:"params,omitempty"`
	Connections map[string]any `json:"connections,omitempty" yaml:"connections,omitempty"`
	UI          map[string]any `json:"ui,omitempty" yaml:"ui,omitempty"`
	AppSpec     *AppSpec       `json:"app,omitempty" yaml:"app,omitempty,omitempty"`
}

type AppSpec struct {
	Envs     map[string]string `json:"envs" yaml:"envs"`
	Policies []string          `json:"policies" yaml:"policies"`
	Secrets  map[string]Secret `json:"secrets" yaml:"secrets"`
}

type Secret struct {
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	JSON        bool   `json:"json,omitempty" yaml:"json,omitempty"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

func (b *Bundle) IsInfrastructure() bool {
	return b.Type == "bundle" || b.Type == "infrastructure"
}

func (b *Bundle) IsApplication() bool {
	return b.Type == "application"
}

func Unmarshal(readDirectory string) (*Bundle, error) {
	unmarshalledBundle := &Bundle{}
	if err := files.Read(path.Join(readDirectory, "massdriver.yaml"), unmarshalledBundle); err != nil {
		return nil, err
	}

	if unmarshalledBundle.Access != "" {
		fmt.Println(prettylogs.Orange("Warning: the 'access' field in massdriver.yaml is no longer supported and should be removed."))
	}
	if unmarshalledBundle.Type != "infrastructure" && unmarshalledBundle.Type != "application" {
		fmt.Println(prettylogs.Orange("Warning: the 'type' field in massdriver.yaml should be either 'infrastructure' or 'application'. This will be enforced in a future release."))
	}

	applyAppBlockDefaults(unmarshalledBundle)
	applyStepDefaults(unmarshalledBundle)

	// This looks weird but we have to be careful we don't overwrite things that do exist in the bundle file
	if unmarshalledBundle.Connections == nil {
		unmarshalledBundle.Connections = make(map[string]any)
	}

	if unmarshalledBundle.Connections["properties"] == nil {
		unmarshalledBundle.Connections["properties"] = make(map[string]any)
	}

	if unmarshalledBundle.Artifacts == nil {
		unmarshalledBundle.Artifacts = make(map[string]any)
	}

	if unmarshalledBundle.Artifacts["properties"] == nil {
		unmarshalledBundle.Artifacts["properties"] = make(map[string]any)
	}

	return unmarshalledBundle, nil
}

func applyAppBlockDefaults(b *Bundle) {
	if b.AppSpec != nil {
		if b.AppSpec.Envs == nil {
			b.AppSpec.Envs = map[string]string{}
		}
		if b.AppSpec.Policies == nil {
			b.AppSpec.Policies = []string{}
		}
		if b.AppSpec.Secrets == nil {
			b.AppSpec.Secrets = map[string]Secret{}
		}
	}
}

func applyStepDefaults(b *Bundle) {
	if len(b.Steps) == 0 {
		msg := fmt.Sprintf(`%s: No steps defined in massdriver.yaml, defaulting to Terraform provisioner. This will be deprecated in a future release. To avoid this warning, please add the following to massdriver.yaml:
steps:
  - path: src
    provisioner: terraform`, prettylogs.Orange("Warning"))
		fmt.Println(msg + "\n")
		b.Steps = append(b.Steps, Step{Path: "src", Provisioner: "terraform"})
	}
}

func parseMetadataSchema() map[string]any {
	metadataBytes, err := embedFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return nil
	}

	var metadata map[string]any
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil
	}

	return metadata
}
