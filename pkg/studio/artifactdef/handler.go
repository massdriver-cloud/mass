package artifactdef

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"gopkg.in/yaml.v3"
)

// Handler handles artifact definition-specific API requests
type Handler struct {
	defPath  string
	mdClient *client.Client
}

// NewHandler creates a new artifact definition handler for the given directory
func NewHandler(dir string, mdClient *client.Client) (*Handler, error) {
	yamlPath := filepath.Join(dir, "massdriver.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		return nil, fmt.Errorf("massdriver.yaml not found in %s: %w", dir, err)
	}

	return &Handler{defPath: dir, mdClient: mdClient}, nil
}

// GetDefinition returns the raw artifact definition content
func (h *Handler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	yamlPath := filepath.Join(h.defPath, "massdriver.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse YAML to return as JSON
	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse YAML: %v", err), http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(raw)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

// GetSchema returns the built schema with $md block (API format)
func (h *Handler) GetSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	yamlPath := filepath.Join(h.defPath, "massdriver.yaml")
	built, err := definition.Build(yamlPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to build schema: %v", err), http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(built)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

// SaveDefinition saves an artifact definition to the massdriver.yaml file
func (h *Handler) SaveDefinition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the incoming JSON
	var incoming map[string]any
	if err := json.Unmarshal(body, &incoming); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Write as YAML
	yamlPath := filepath.Join(h.defPath, "massdriver.yaml")
	if err := files.Write(yamlPath, incoming); err != nil {
		http.Error(w, fmt.Sprintf("failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"status":"saved"}`))
	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

// CreateDefinition creates a new artifact definition in the specified path
func CreateDefinition(w http.ResponseWriter, r *http.Request, basePath string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the incoming JSON - expects {path: string, definition: object}
	var request struct {
		Path       string         `json:"path"`
		Definition map[string]any `json:"definition"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}

	if request.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	// Resolve path relative to base
	targetDir := request.Path
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(basePath, request.Path)
	}

	// Validate target is within base path (prevent path traversal)
	absBase, _ := filepath.Abs(basePath)
	absTarget, _ := filepath.Abs(targetDir)
	if !isSubPath(absBase, absTarget) {
		http.Error(w, "invalid path: must be within studio directory", http.StatusBadRequest)
		return
	}

	// Create directory if needed
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if file already exists
	yamlPath := filepath.Join(targetDir, "massdriver.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		http.Error(w, "massdriver.yaml already exists at this path", http.StatusConflict)
		return
	}

	// Write the definition
	if err := files.Write(yamlPath, request.Definition); err != nil {
		http.Error(w, fmt.Sprintf("failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]string{
		"status": "created",
		"path":   targetDir,
	}
	out, _ := json.Marshal(response)
	_, err = w.Write(out)
	if err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

// isSubPath checks if target is under base directory
func isSubPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	// Check that we don't go up directories
	return !filepath.IsAbs(rel) && rel != ".." && !startsWithDotDot(rel)
}

func startsWithDotDot(path string) bool {
	return len(path) >= 2 && path[0:2] == ".."
}
