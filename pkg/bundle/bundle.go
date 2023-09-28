package bundle

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

const (
	ParamsFile     = "_params.auto.tfvars.json"
	ConnsFile      = "_connections.auto.tfvars.json"
	allowedMethods = "OPTIONS, GET, POST"
)

type Handler struct {
	Bundle    Bundle
	fs        afero.Fs
	bundleDir string
}
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

	err = checkForOperatorGuideAndSetValue(srcDir, &body, fs)

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

func checkForOperatorGuideAndSetValue(path string, body *restclient.PublishPost, fs afero.Fs) error {
	pathsToCheck := []string{"operator.mdx", "operator.md"}

	for _, fileName := range pathsToCheck {
		_, err := fs.Stat(filepath.Join(path, fileName))

		if err != nil {
			continue
		}

		content, err := afero.ReadFile(fs, filepath.Join(path, fileName))

		if err != nil {
			return fmt.Errorf("error reading %s", fileName)
		}

		body.OperatorGuide = content
	}

	return nil
}

func UnmarshalBundle(readDirectory string, fs afero.Fs) (*Bundle, error) {
	file, err := afero.ReadFile(fs, path.Join(readDirectory, "massdriver.yaml"))
	if err != nil {
		return nil, err
	}

	unmarshalledBundle := &Bundle{}

	err = yaml.Unmarshal(file, unmarshalledBundle)
	if err != nil {
		return nil, err
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

func NewHandler(dir string) (*Handler, error) {
	fs := afero.NewOsFs()
	bundle, err := UnmarshalBundle(dir, fs)
	if err != nil {
		return nil, err
	}
	ApplyAppBlockDefaults(bundle)
	return &Handler{Bundle: *bundle, fs: fs, bundleDir: dir}, nil
}

func (b *Handler) GetSecrets(w http.ResponseWriter, _ *http.Request) {
	out, err := json.Marshal(b.Bundle.AppSpec.Secrets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error(err.Error())
	}
}

func (b *Handler) Connections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		b.getConnections(w)
	case http.MethodPost:
		b.postConnections(w, r)
	case http.MethodOptions:
		b.options(w, r)
	default:
		w.Header().Add("Allow", allowedMethods)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (b *Handler) getConnections(w http.ResponseWriter) {
	f, err := afero.ReadFile(b.fs, path.Join(b.bundleDir, "src", ConnsFile))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(f)
	if err != nil {
		slog.Error(err.Error())
	}
}

func (b *Handler) postConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Debug("Error reading payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	connMap := make(map[string]any)

	// We have to go through the unmarshal/marshal dance to ensure
	// we keep the formatting in the final file. If the json payload
	// is a single line that would end up back in the file and make
	// it unreadable.
	err = json.Unmarshal(conns, &connMap)
	if err != nil {
		slog.Debug("Error unmarshalling payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bytes, err := json.MarshalIndent(connMap, "", "    ")
	if err != nil {
		slog.Debug("Error marshalling payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = afero.WriteFile(b.fs, path.Join(b.bundleDir, "src", ConnsFile), bytes, 0755)
	if err != nil {
		slog.Debug("Error writing file", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (b *Handler) options(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	headers["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
	headers["Access-Control-Allow-Methods"] = []string{allowedMethods}
	w.WriteHeader(http.StatusOK)
}
