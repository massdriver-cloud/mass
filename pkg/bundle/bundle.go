package bundle

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

//go:embed schemas/metadata-schema.json
var embedFS embed.FS

var MetadataSchema = parseMetadataSchema()

const (
	ParamsFile = "_params.auto.tfvars.json"
	ConnsFile  = "_connections.auto.tfvars.json"
)

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
	Connections Connections            `json:"connections" yaml:"connections"`
	UI          map[string]interface{} `json:"ui" yaml:"ui"`
	AppSpec     *AppSpec               `json:"app,omitempty" yaml:"app,omitempty"`
}

type Connections = map[string]any

type AppSpec struct {
	Envs     map[string]string `json:"envs" yaml:"envs"`
	Policies []string          `json:"policies" yaml:"policies"`
	Secrets  map[string]Secret `json:"secrets" yaml:"secrets"`
}

type Secret struct {
	Required    bool   `json:"required" yaml:"required"`
	JSON        bool   `json:"json" yaml:"json"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

func (b *Bundle) GenerateBundlePublishBody(srcDir string) (restclient.PublishPost, error) {
	var body restclient.PublishPost

	body.Name = b.Name
	body.Description = b.Description
	body.Type = b.Type
	body.SourceURL = b.SourceURL
	body.Access = b.Access
	body.ArtifactsSchema = b.Artifacts
	body.ConnectionsSchema = b.Connections
	body.ParamsSchema = b.Params
	body.UISchema = b.UI

	var appSpec map[string]interface{}
	marshalledAppSpec, err := json.Marshal(b.AppSpec)
	if err != nil {
		return restclient.PublishPost{}, err
	}
	err = json.Unmarshal(marshalledAppSpec, &appSpec)
	if err != nil {
		fmt.Println(err)
		return restclient.PublishPost{}, err
	}
	body.AppSpec = appSpec

	var bundleSpec map[string]interface{}
	marshalledBundleSpec, err := json.Marshal(b)
	if err != nil {
		return restclient.PublishPost{}, err
	}
	err = json.Unmarshal(marshalledBundleSpec, &bundleSpec)
	if err != nil {
		fmt.Println(err)
		return restclient.PublishPost{}, err
	}
	body.Spec = bundleSpec

	err = checkForOperatorGuideAndSetValue(srcDir, &body)

	if err != nil {
		return restclient.PublishPost{}, err
	}

	return body, nil
}

func (b *Bundle) IsInfrastructure() bool {
	return b.Type == "bundle" || b.Type == "infrastructure"
}

func (b *Bundle) IsApplication() bool {
	return b.Type == "application"
}

func checkForOperatorGuideAndSetValue(path string, body *restclient.PublishPost) error {
	pathsToCheck := []string{"operator.mdx", "operator.md"}

	for _, fileName := range pathsToCheck {
		_, err := os.Stat(filepath.Join(path, fileName))

		if err != nil {
			continue
		}

		content, err := os.ReadFile(filepath.Join(path, fileName))

		if err != nil {
			return fmt.Errorf("error reading %s", fileName)
		}

		body.OperatorGuide = content
	}

	return nil
}

func Unmarshal(readDirectory string) (*Bundle, error) {
	unmarshalledBundle := &Bundle{}
	if err := files.Read(path.Join(readDirectory, "massdriver.yaml"), unmarshalledBundle); err != nil {
		return nil, err
	}

	return unmarshalledBundle, nil
}

func UnmarshalAndApplyDefaults(readDirectory string) (*Bundle, error) {
	unmarshalledBundle, err := Unmarshal(readDirectory)
	if err != nil {
		return nil, err
	}

	if unmarshalledBundle.IsApplication() {
		ApplyAppBlockDefaults(unmarshalledBundle)
	}

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

func ApplyAppBlockDefaults(b *Bundle) {
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
	if b.Steps == nil || len(b.Steps) == 0 {
		msg := fmt.Sprintf(`%s: No steps defined in massdriver.yaml, defaulting to Terraform provisioner. This will be deprecated in a future release. To avoid this warning, please add the following to massdriver.yaml:
steps:
    path: src
    provisioner: terraform`, prettylogs.Orange("Warning"))
		fmt.Println(msg + "\n")
		b.Steps = append(b.Steps, Step{Path: "src", Provisioner: "terraform"})
	}
}

func parseMetadataSchema() map[string]interface{} {
	metadataBytes, err := embedFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return nil
	}

	var metadata map[string]interface{}
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil
	}

	return metadata
}
