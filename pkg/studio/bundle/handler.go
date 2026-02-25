package bundle

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

const (
	allowedMethods = "OPTIONS, POST"
	paramsFile     = "_params.auto.tfvars.json"
)

// Handler handles bundle-specific API requests
type Handler struct {
	parsedBundle bundle.Bundle
	bundleDir    string
	mdClient     *client.Client
}

// NewHandler creates a new bundle handler for the given directory
func NewHandler(dir string, mdClient *client.Client) (*Handler, error) {
	b, err := bundle.Unmarshal(dir)
	if err != nil {
		return nil, err
	}

	return &Handler{parsedBundle: *b, bundleDir: dir, mdClient: mdClient}, nil
}

// GetBundle returns the bundle data
func (h *Handler) GetBundle(w http.ResponseWriter, _ *http.Request) {
	out, err := json.Marshal(h.parsedBundle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error(err.Error())
	}
}

// GetSecrets returns the secrets from the bundle
func (h *Handler) GetSecrets(w http.ResponseWriter, _ *http.Request) {
	var out []byte
	var err error
	if h.parsedBundle.AppSpec == nil || len(h.parsedBundle.AppSpec.Secrets) == 0 {
		out = []byte("{}")
	} else {
		out, err = json.Marshal(h.parsedBundle.AppSpec.Secrets)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error(err.Error())
	}
}

// GetEnvironmentVariables returns the parsed env vars from an application bundle
func (h *Handler) GetEnvironmentVariables(w http.ResponseWriter, _ *http.Request) {
	var out []byte
	if h.parsedBundle.AppSpec == nil || len(h.parsedBundle.AppSpec.Envs) == 0 {
		out = []byte("{}")
	} else {
		paramsAndConnections, err := h.getUserInput()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		result := bundle.ParseEnvironmentVariables(paramsAndConnections, h.parsedBundle.AppSpec.Envs)

		out, err = json.Marshal(result)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(out)
	if err != nil {
		slog.Error(err.Error())
	}
}

// Params writes and fetches current parameters to file on demand
func (h *Handler) Params(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodOptions && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodOptions {
		h.options(w, r)
		return
	}

	if r.Method == http.MethodGet {
		content, err := os.ReadFile(path.Join(h.bundleDir, "src", paramsFile))
		if err != nil {
			slog.Debug("Error reading params file", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = w.Write(content)
		if err != nil {
			slog.Warn("failed to write response", "error", err)
			return
		}
		return
	}

	params, err := io.ReadAll(r.Body)

	if err != nil {
		slog.Debug("Error reading payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	payload := make(map[string]any)

	err = json.Unmarshal(params, &payload)

	if err != nil {
		slog.Debug("Error unmarshalling payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = ReconcileParams(h.bundleDir, payload); err != nil {
		slog.Debug("Error writing params contents", "error", err)
		http.Error(w, "unable to write params file", http.StatusBadRequest)
		return
	}

	_, err = w.Write([]byte("{}"))
	if err != nil {
		slog.Warn("failed to write response", "error", err)
	}
}

func (h *Handler) getUserInput() (map[string]any, error) {
	output := make(map[string]any)

	conns, err := h.readFileAndUnmarshal(bundle.ConnsFile)

	if err != nil {
		return output, err
	}

	params, err := h.readFileAndUnmarshal(bundle.ParamsFile)

	if err != nil {
		return output, err
	}

	output["connections"] = conns
	output["params"] = params

	return output, nil
}

func (h *Handler) readFileAndUnmarshal(readPath string) (map[string]any, error) {
	output := make(map[string]any)

	file, err := os.ReadFile(path.Join(h.bundleDir, "src", readPath))

	if err != nil {
		return output, err
	}

	err = json.Unmarshal(file, &output)

	return output, err
}

// Build rebuilds the bundle
func (h *Handler) Build(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Add("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	unmarshalledBundle, err := bundle.Unmarshal(h.bundleDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = unmarshalledBundle.Build(h.bundleDir, h.mdClient); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// After rebuilding the bundle, load it back onto the handler
	b, err := bundle.Unmarshal(h.bundleDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.parsedBundle = *b
	w.WriteHeader(http.StatusOK)
}

// Connections handles GET/POST for bundle connections
func (h *Handler) Connections(w http.ResponseWriter, r *http.Request) {
	allowedMethods := "GET, POST"
	switch r.Method {
	case http.MethodGet:
		h.getConnections(w)
	case http.MethodPost:
		h.postConnections(w, r)
	default:
		w.Header().Add("Allow", allowedMethods)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *Handler) getConnections(w http.ResponseWriter) {
	f, err := os.ReadFile(path.Join(h.bundleDir, "src", bundle.ConnsFile))
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

func (h *Handler) postConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Debug("Error reading payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	connMap := make(map[string]any)

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

	// #nosec G306
	err = os.WriteFile(path.Join(h.bundleDir, "src", bundle.ConnsFile), bytes, 0644)
	if err != nil {
		slog.Debug("Error writing file", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (h *Handler) options(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	headers["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
	headers["Access-Control-Allow-Methods"] = []string{allowedMethods}
	w.WriteHeader(http.StatusOK)
}

// ReconcileParams reads the params file keeping the md_metadata field intact,
// adds the incoming params, and writes the file back out.
func ReconcileParams(baseDir string, params map[string]any) error {
	paramPath := path.Join(baseDir, "src", paramsFile)

	fileParams := make(map[string]any)
	err := files.Read(paramPath, &fileParams)
	if err != nil {
		return err
	}

	combinedParams := make(map[string]any)
	if v, ok := fileParams["md_metadata"]; ok {
		combinedParams["md_metadata"] = v
	}

	for k, v := range params {
		combinedParams[k] = v
	}

	return files.Write(paramPath, combinedParams)
}
