package templates

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/massdriver-cloud/mass/pkg/config"
)

type BundleTemplateCache struct {
	TemplatePath string
}

// RefreshTemplates is a no-op since templates are now managed locally
func (b *BundleTemplateCache) RefreshTemplates() error {
	return nil
}

// ListTemplates lists all templates available in the local templates directory.
// Templates are expected at PATH/{template}/massdriver.yaml
func (b *BundleTemplateCache) ListTemplates() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(b.TemplatePath, "*", "massdriver.yaml"))
	if err != nil {
		return nil, err
	}

	templates := make([]string, 0, len(matches))
	for _, match := range matches {
		// Extract template name from path
		// PATH/{template}/massdriver.yaml -> {template}
		relPath := strings.TrimPrefix(match, b.TemplatePath+string(filepath.Separator))
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 1 {
			templates = append(templates, parts[0])
		}
	}

	return templates, nil
}

// GetTemplatePath returns the path to the template directory
func (b *BundleTemplateCache) GetTemplatePath() (string, error) {
	return b.TemplatePath, nil
}

// RenderTemplate copies the template to the output directory and renders massdriver.yaml with user-supplied values
func (b *BundleTemplateCache) RenderTemplate(data *TemplateData) error {
	fileManager := &fileManager{
		readDirectory:         path.Join(b.TemplatePath, data.TemplateName),
		writeDirectory:        data.OutputDir,
		templateData:          data,
		templateRootDirectory: b.TemplatePath,
	}

	return fileManager.CopyTemplate()
}

// NewBundleTemplateCache creates a new BundleTemplateCache using the configured templates path.
// The templates path is determined by:
// 1. MD_TEMPLATES_PATH environment variable
// 2. templates_path from ~/.config/massdriver/config.yaml
func NewBundleTemplateCache() (TemplateCache, error) {
	templatePath, err := config.GetTemplatesPath()
	if err != nil {
		return nil, err
	}

	return &BundleTemplateCache{
		TemplatePath: templatePath,
	}, nil
}
