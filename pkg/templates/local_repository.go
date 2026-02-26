package templates

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// ErrTemplatesPathNotConfigured is returned when templates path is not set
var ErrTemplatesPathNotConfigured = errors.New("templates path not configured: set MASSDRIVER_TEMPLATES_PATH environment variable or templates_path in profile in ~/.config/massdriver/config.yaml. See https://docs.massdriver.cloud/guides/bundle-templates for more info")

// LocalRepository implements Repository using the local filesystem
type LocalRepository struct {
	TemplatePath string
}

// List returns all templates available in the local templates directory.
// Templates are expected at PATH/{template}/massdriver.yaml
func (r *LocalRepository) List() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(r.TemplatePath, "*", "massdriver.yaml"))
	if err != nil {
		return nil, err
	}

	templates := make([]string, 0, len(matches))
	for _, match := range matches {
		// Extract template name from path
		// PATH/{template}/massdriver.yaml -> {template}
		relPath := strings.TrimPrefix(match, r.TemplatePath+string(filepath.Separator))
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 1 {
			templates = append(templates, parts[0])
		}
	}

	return templates, nil
}

// Path returns the path to the template directory
func (r *LocalRepository) Path() (string, error) {
	return r.TemplatePath, nil
}

// Render copies the template to the output directory and renders massdriver.yaml with user-supplied values
func (r *LocalRepository) Render(data *TemplateData) error {
	fileManager := &fileManager{
		readDirectory:         path.Join(r.TemplatePath, data.TemplateName),
		writeDirectory:        data.OutputDir,
		templateData:          data,
		templateRootDirectory: r.TemplatePath,
	}

	return fileManager.CopyTemplate()
}

// NewRepository creates a new LocalRepository using the configured templates path.
// The templates path is determined by:
// 1. MASSDRIVER_TEMPLATES_PATH environment variable
// 2. templates_path from profile in ~/.config/massdriver/config.yaml
func NewRepository() (Repository, error) {
	cfg, err := sdkconfig.Get()
	if err != nil {
		return nil, ErrTemplatesPathNotConfigured
	}

	if cfg.TemplatesPath == "" {
		return nil, ErrTemplatesPathNotConfigured
	}

	return &LocalRepository{
		TemplatePath: cfg.TemplatesPath,
	}, nil
}
