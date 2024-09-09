package bundle

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

//go:embed schemas/metadata-schema.json
var embedFS embed.FS

var MetadataSchema = parseMetadataSchema()

const (
	ParamsFile = "_params.auto.tfvars.json"
	ConnsFile  = "_connections.auto.tfvars.json"
)

func (b *Bundle) GenerateBundlePublishBody(srcDir string) (restclient.PublishPost, error) {
	var body restclient.PublishPost

	body.Name = b.Name
	body.Description = b.Description
	body.Type = b.Type
	body.SourceURL = b.SourceURL
	body.Access = b.Access
	body.ArtifactsSchema = b.Artifacts.ToMap()
	body.ConnectionsSchema = b.Connections.ToMap()
	body.ParamsSchema = b.Params.ToMap()
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

	err = checkForOperatorGuideAndSetValue(srcDir, &body)

	if err != nil {
		return restclient.PublishPost{}, err
	}

	return body, nil
}

func (b *Bundle) IsInfrastructure() bool {
	// a Deprecation warning is printed in the bundle parse function
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

	// This looks weird but we have to be careful we don't overwrite things that do exist in the bundle file
	if unmarshalledBundle.Params == nil {
		unmarshalledBundle.Params = &Schema{}
	}
	if unmarshalledBundle.Params.Properties == nil {
		unmarshalledBundle.Params.Properties = make(map[string]*Schema)
	}

	if unmarshalledBundle.Connections == nil {
		unmarshalledBundle.Connections = &Schema{}
	}
	if unmarshalledBundle.Connections.Properties == nil {
		unmarshalledBundle.Connections.Properties = make(map[string]*Schema)
	}

	if unmarshalledBundle.Artifacts == nil {
		unmarshalledBundle.Artifacts = &Schema{}
	}
	if unmarshalledBundle.Artifacts.Properties == nil {
		unmarshalledBundle.Artifacts.Properties = make(map[string]*Schema)
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

func parseMetadataSchema() *Schema {
	metadataBytes, err := embedFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return nil
	}

	metadata := new(Schema)
	err = json.Unmarshal(metadataBytes, metadata)
	if err != nil {
		return nil
	}

	return metadata
}
