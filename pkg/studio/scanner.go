package studio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/massdriver-cloud/mass/pkg/files"
)

// ItemType represents the type of a discovered massdriver.yaml item
type ItemType string

const (
	ItemTypeBundle             ItemType = "bundle"
	ItemTypeArtifactDefinition ItemType = "artifact-definition"
)

// StudioItem represents a discovered massdriver.yaml file
type StudioItem struct {
	Path         string    `json:"path"`         // Absolute path to the directory containing massdriver.yaml
	Type         ItemType  `json:"type"`         // bundle or artifact-definition
	Name         string    `json:"name"`         // Name from the massdriver.yaml
	Description  string    `json:"description"`  // Description (for bundles)
	LastModified time.Time `json:"lastModified"` // Last modification time of massdriver.yaml
	Error        string    `json:"error"`        // Error message if parsing failed
}

// rawMassdriverYAML is used for initial parsing to get basic info
type rawMassdriverYAML struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ScanDirectory recursively walks the directory tree looking for massdriver.yaml files
// and categorizes them as bundles or artifact definitions
func ScanDirectory(rootDir string) ([]StudioItem, error) {
	var items []StudioItem

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if d.IsDir() && len(d.Name()) > 1 && d.Name()[0] == '.' {
			return filepath.SkipDir
		}

		// Look for massdriver.yaml files
		if !d.IsDir() && d.Name() == "massdriver.yaml" {
			item, scanErr := scanItem(path)
			if scanErr != nil {
				// Still add the item but with an error
				items = append(items, StudioItem{
					Path:  filepath.Dir(path),
					Error: scanErr.Error(),
				})
			} else {
				items = append(items, *item)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return items, nil
}

// scanItem parses a single massdriver.yaml file and returns a StudioItem
func scanItem(yamlPath string) (*StudioItem, error) {
	info, err := os.Stat(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	itemType, err := DetectItemType(yamlPath)
	if err != nil {
		return nil, err
	}

	var raw rawMassdriverYAML
	if err := files.Read(yamlPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse massdriver.yaml: %w", err)
	}

	return &StudioItem{
		Path:         filepath.Dir(yamlPath),
		Type:         itemType,
		Name:         raw.Name,
		Description:  raw.Description,
		LastModified: info.ModTime(),
	}, nil
}

// DetectItemType reads a massdriver.yaml file and determines if it's a bundle or artifact definition
// based on the presence of specific fields:
// - Has "steps" field → Bundle
// - Has "schema" field that is a map (not string) → Artifact Definition
// - Has "params" or "connections" or "artifacts" → Bundle (legacy detection)
func DetectItemType(yamlPath string) (ItemType, error) {
	var raw map[string]any
	if err := files.Read(yamlPath, &raw); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check for steps field - indicates a bundle
	if _, hasSteps := raw["steps"]; hasSteps {
		return ItemTypeBundle, nil
	}

	// Check for schema field - but distinguish between bundle and artifact definition
	// Bundles have schema as a string (e.g., "draft-07")
	// Artifact definitions have schema as a map (the actual JSON schema)
	if schema, hasSchema := raw["schema"]; hasSchema {
		// If schema is a map, it's an artifact definition
		if _, isMap := schema.(map[string]any); isMap {
			return ItemTypeArtifactDefinition, nil
		}
		// If schema is a string (like "draft-07"), it's likely a bundle
		// Fall through to bundle detection
	}

	// Bundle detection: if it has params/connections/artifacts, it's a bundle
	if _, hasParams := raw["params"]; hasParams {
		return ItemTypeBundle, nil
	}
	if _, hasConnections := raw["connections"]; hasConnections {
		return ItemTypeBundle, nil
	}
	if _, hasArtifacts := raw["artifacts"]; hasArtifacts {
		return ItemTypeBundle, nil
	}

	return "", errors.New("unable to determine type: file has neither 'steps' (bundle) nor 'schema' map (artifact definition)")
}

// FilterByType returns items of a specific type
func FilterByType(items []StudioItem, itemType ItemType) []StudioItem {
	var filtered []StudioItem
	for _, item := range items {
		if item.Type == itemType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
