// Package resourcetype provides utilities for reading, building, and publishing resource types.
package resourcetype

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ArtifactType is the OCI artifact-type media type for resource types.
const ArtifactType = "application/vnd.massdriver.resource-type.v1+json"

// MassdriverYAML represents the structure of a massdriver.yaml resource type file.
// This is an experimental format that provides a more ergonomic authoring experience.
type MassdriverYAML struct {
	Name    string         `yaml:"name"`
	Version string         `yaml:"version,omitempty"`
	Label   string         `yaml:"label,omitempty"`
	Icon    string         `yaml:"icon,omitempty"`
	UI      *UIConfig      `yaml:"ui,omitempty"`
	Exports []ExportConfig `yaml:"exports,omitempty"`
	Schema  map[string]any `yaml:"schema"`
}

// UIConfig represents the UI configuration section
type UIConfig struct {
	ConnectionOrientation   string              `yaml:"connectionOrientation,omitempty"`
	EnvironmentDefaultGroup string              `yaml:"environmentDefaultGroup,omitempty"`
	Instructions            []InstructionConfig `yaml:"instructions,omitempty"`
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

// ReadConfig parses a massdriver.yaml without dereferencing or building it.
func ReadConfig(path string) (*MassdriverYAML, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read massdriver.yaml: %w", err)
	}

	var config MassdriverYAML
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse massdriver.yaml: %w", err)
	}

	return &config, nil
}

// Build converts a massdriver.yaml into the format the API expects.
func Build(path string) (map[string]any, error) {
	config, err := ReadConfig(path)
	if err != nil {
		return nil, err
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

// IsMassdriverYAMLResourceType checks if the given path is a massdriver.yaml
// file that should be treated as a resource type in the experimental format.
func IsMassdriverYAMLResourceType(path string) bool {
	return filepath.Base(path) == "massdriver.yaml"
}
