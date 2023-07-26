package bundle

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
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
	Connections map[string]interface{} `json:"connections" yaml:"connections"`
	UI          map[string]interface{} `json:"ui" yaml:"ui"`
	AppSpec     *AppSpec               `json:"app,omitempty" yaml:"app,omitempty"`
}

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

type docAdder func(path string, body *restclient.PublishPost, fs afero.Fs) error

func (b *Bundle) GenerateBundlePublishBody(srcDir string, fs afero.Fs) (restclient.PublishPost, error) {
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

	err = checkForGuidesAndSetValue(srcDir, &body, fs)

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

func checkForGuidesAndSetValue(path string, body *restclient.PublishPost, fs afero.Fs) error {
	documentFetchers := []docAdder{checkForOperatorGuideAndSetValue, checkForRunbookAndSetValue}

	for _, fetcher := range documentFetchers {
		err := fetcher(path, body, fs)

		if err != nil {
			return err
		}
	}

	return nil
}

func checkForOperatorGuideAndSetValue(path string, body *restclient.PublishPost, fs afero.Fs) error {
	pathsToCheck := []string{"operator.mdx", "operator.md"}
	content, err := checkFileAndReadContents(path, pathsToCheck, fs)

	if err != nil {
		return err
	}

	body.OperatorGuide = content
	return nil
}

func checkForRunbookAndSetValue(path string, body *restclient.PublishPost, fs afero.Fs) error {
	pathsToCheck := []string{"runbook.mdx", "runbook.md"}
	content, err := checkFileAndReadContents(path, pathsToCheck, fs)

	if err != nil {
		return err
	}

	body.Runbook = content
	return nil
}

func checkFileAndReadContents(path string, pathsToCheck []string, fs afero.Fs) ([]byte, error) {
	var content []byte

	for _, fileName := range pathsToCheck {
		_, err := fs.Stat(filepath.Join(path, fileName))

		if err != nil {
			continue
		}

		content, err = afero.ReadFile(fs, filepath.Join(path, fileName))

		if err != nil {
			return content, fmt.Errorf("error reading %s", fileName)
		}

		return content, nil
	}

	return content, nil
}
