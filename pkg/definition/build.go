package definition

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MassdriverYAML represents the structure of a massdriver.yaml artifact definition file.
// This is an experimental format that provides a more ergonomic authoring experience.
type MassdriverYAML struct {
	Name    string         `yaml:"name"`
	Label   string         `yaml:"label"`
	Icon    string         `yaml:"icon"`
	UI      *UIConfig      `yaml:"ui"`
	Exports []ExportConfig `yaml:"exports"`
	Schema  map[string]any `yaml:"schema"`
}

// UIConfig represents the UI configuration section
type UIConfig struct {
	ConnectionOrientation   string              `yaml:"connectionOrientation"`
	EnvironmentDefaultGroup string              `yaml:"environmentDefaultGroup"`
	Instructions            []InstructionConfig `yaml:"instructions"`
}

// InstructionConfig represents an instruction file reference
type InstructionConfig struct {
	Label string `yaml:"label"`
	Path  string `yaml:"path"`
}

// ExportConfig represents an export template configuration
type ExportConfig struct {
	DownloadButtonText string `yaml:"downloadButtonText"`
	FileFormat         string `yaml:"fileFormat"`
	TemplatePath       string `yaml:"templatePath"`
	TemplateLang       string `yaml:"templateLang"`
}

// Build reads a massdriver.yaml file and builds it into the artifact definition
// format expected by the Massdriver API.
func Build(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read massdriver.yaml: %w", err)
	}

	var config MassdriverYAML
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse massdriver.yaml: %w", err)
	}

	baseDir := filepath.Dir(path)

	// Build the $md block
	mdBlock := map[string]any{
		"name":  config.Name,
		"label": config.Label,
		"icon":  config.Icon,
	}

	// Process UI configuration
	if config.UI != nil {
		uiBlock := map[string]any{}

		if config.UI.ConnectionOrientation != "" {
			uiBlock["connectionOrientation"] = config.UI.ConnectionOrientation
		}
		if config.UI.EnvironmentDefaultGroup != "" {
			uiBlock["environmentDefaultGroup"] = config.UI.EnvironmentDefaultGroup
		}

		// Process instructions
		instructions := []map[string]any{}
		for _, instruction := range config.UI.Instructions {
			instructionPath := filepath.Join(baseDir, instruction.Path)
			instructionContent, err := os.ReadFile(instructionPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read instruction file %s: %w", instruction.Path, err)
			}
			instructions = append(instructions, map[string]any{
				"label":   instruction.Label,
				"content": string(instructionContent),
			})
		}
		uiBlock["instructions"] = instructions

		mdBlock["ui"] = uiBlock
	}

	// Process exports
	exports := []map[string]any{}
	for _, export := range config.Exports {
		templatePath := filepath.Join(baseDir, export.TemplatePath)
		templateContent, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read export template %s: %w", export.TemplatePath, err)
		}
		exports = append(exports, map[string]any{
			"downloadButtonText": export.DownloadButtonText,
			"fileFormat":         export.FileFormat,
			"template":           string(templateContent),
			"templateLang":       export.TemplateLang,
		})
	}
	mdBlock["export"] = exports

	// Build the final structure: merge $md with schema
	result := map[string]any{
		"$md": mdBlock,
	}

	// Merge schema into result
	for key, value := range config.Schema {
		result[key] = value
	}

	return result, nil
}

// IsMassdriverYAMLArtifactDefinition checks if the given path is a massdriver.yaml
// file that should be treated as an artifact definition in the experimental format.
func IsMassdriverYAMLArtifactDefinition(path string) bool {
	return filepath.Base(path) == "massdriver.yaml"
}
