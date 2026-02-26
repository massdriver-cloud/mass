package templates

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

var ErrNotConfigured = errors.New("templates path not configured: set MASSDRIVER_TEMPLATES_PATH environment variable or templates_path in profile in ~/.config/massdriver/config.yaml. See https://docs.massdriver.cloud/guides/bundle-templates for more info")

type Templates struct {
	Path string
}

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

type Connection struct {
	Name               string `json:"name"`
	ArtifactDefinition string `json:"artifact_definition"`
}

func New() (*Templates, error) {
	cfg, err := sdkconfig.Get()
	if err != nil {
		return nil, ErrNotConfigured
	}
	if cfg.TemplatesPath == "" {
		return nil, ErrNotConfigured
	}
	return &Templates{Path: cfg.TemplatesPath}, nil
}

func (t *Templates) List() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(t.Path, "*", "massdriver.yaml"))
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		relPath := strings.TrimPrefix(match, t.Path+string(filepath.Separator))
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 1 {
			result = append(result, parts[0])
		}
	}
	return result, nil
}

func (t *Templates) Render(data *TemplateData) error {
	fm := &fileManager{
		readDirectory:         path.Join(t.Path, data.TemplateName),
		writeDirectory:        data.OutputDir,
		templateData:          data,
		templateRootDirectory: t.Path,
	}
	return fm.CopyTemplate()
}
