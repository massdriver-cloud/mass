package bundle

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands/bundle/build"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

const allowedMethods = "OPTIONS, POST"

type Handler struct {
	parsedBundle bundle.Bundle
	bundleDir    string
	mdClient     *client.Client
}

func NewHandler(dir string, mdClient *client.Client) (*Handler, error) {
	b, err := bundle.Unmarshal(dir)
	if err != nil {
		return nil, err
	}
	bundle.ApplyAppBlockDefaults(b)

	return &Handler{parsedBundle: *b, bundleDir: dir, mdClient: mdClient}, nil
}

// GetSecrets returns the secrets from the bundle
//
//	@Summary		Get bundle secrets
//	@Description	Get bundle secrets
//	@ID				get-bundle-secrets
//	@Produce		json
//	@Success		200	{object}	bundle.AppSpec.Secrets
//	@Router			/bundle/secrets [get]
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
//
//	@Summary		Get parsed env vars
//	@Description	Get parsed env vars
//	@ID				get-environment-variables
//	@Produce		json
//	@Success		200	{object}	map[string]bundle.ParsedEnvironmentVariable
//	@Router			/bundle/envs [get]
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
//
//	@Summary		Write and fetch currently set params to vars file
//	@Description	Allows users to set their params before deployment for easy recall
//	@ID				params
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Router			/bundle/params [post, get, options]
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

	if err = build.Run(h.bundleDir, unmarshalledBundle, h.mdClient); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// After rebuilding the bundle, load it back onto the handler
	b, err := bundle.Unmarshal(h.bundleDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bundle.ApplyAppBlockDefaults(b)

	h.parsedBundle = *b
	w.WriteHeader(http.StatusOK)
}

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

// getConnections returns the existing connections in the conn file
//
//	@Summary		Get bundle connections
//	@Description	Get bundle connections
//	@ID				get-bundle-connections
//	@Produce		json
//	@Success		200	{object}	bundle.Connections
//	@Router			/bundle/connections [get]
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

// postConnections accepts connections and writes them back to the conn file
//
//	@Summary		Post bundle connections
//	@Description	Post bundle connections
//	@ID				post-bundle-connections
//	@Accept			json
//	@Success		200			{string}	string				"success"
//	@Param			connectons	body		bundle.Connections	true	"Connections"
//	@Router			/bundle/connections [post]
func (h *Handler) postConnections(w http.ResponseWriter, r *http.Request) {
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
