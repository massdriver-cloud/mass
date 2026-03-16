package templates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

const envTemplatesPath = "MASSDRIVER_TEMPLATES_PATH"

// ErrNotConfigured is returned when the templates path has not been set via env var or config file.
var ErrNotConfigured = errors.New("templates path not configured: set MASSDRIVER_TEMPLATES_PATH environment variable or templates_path in profile in ~/.config/massdriver/config.yaml. See https://docs.massdriver.cloud/guides/bundle-templates for more info")

// TemplateData holds values used when rendering a bundle template.
type TemplateData struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Location           string            `json:"location"`
	TemplateName       string            `json:"templateName"`
	OutputDir          string            `json:"outputDir"`
	Type               string            `json:"type"`
	Connections        []Connection      `json:"connections"`
	Envs               map[string]string `json:"envs"`
	ParamsSchema       string            `json:"paramsSchema"`
	ExistingParamsPath string            `json:"existingParamsPath"`
	CloudAbbreviation  string            `json:"cloudAbbreviation"`
	RepoName           string            `json:"repoName"`
	RepoNameEncoded    string            `json:"repoNameEncoded"`
}

// Connection represents a bundle connection with a name and artifact definition reference.
type Connection struct {
	Name               string `json:"name"`
	ArtifactDefinition string `json:"artifact_definition"`
}

func getPath() (string, error) {
	// Check env var directly first - allows tests and standalone usage
	if envPath := os.Getenv(envTemplatesPath); envPath != "" {
		return envPath, nil
	}

	// Fall back to SDK config (requires full config with credentials)
	cfg, err := sdkconfig.Get()
	if err == nil && cfg.TemplatesPath != "" {
		return cfg.TemplatesPath, nil
	}

	return "", ErrNotConfigured
}

// List returns the names of all available bundle templates.
func List() ([]string, error) {
	templatesPath, err := getPath()
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(templatesPath, "*", "massdriver.yaml"))
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		relPath := strings.TrimPrefix(match, templatesPath+string(filepath.Separator))
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 1 {
			result = append(result, parts[0])
		}
	}
	return result, nil
}

// Render copies and renders the named template into the output directory specified in data.
func Render(data *TemplateData) error {
	templatesPath, err := getPath()
	if err != nil {
		return err
	}

	fm := &fileManager{
		readDirectory:         filepath.Join(templatesPath, data.TemplateName),
		writeDirectory:        data.OutputDir,
		templateData:          data,
		templateRootDirectory: templatesPath,
	}
	return fm.CopyTemplate()
}
