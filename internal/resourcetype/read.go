package resourcetype

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"gopkg.in/yaml.v3"
)

// Read reads and dereferences a resource type from path, supporting JSON, YAML, and massdriver.yaml formats.
func Read(_ context.Context, mdClient *client.Client, path string) (map[string]any, error) {
	// Check if this is a massdriver.yaml file (experimental resource type format)
	if IsMassdriverYAMLResourceType(path) {
		built, buildErr := Build(path)
		if buildErr != nil {
			return nil, fmt.Errorf("failed to build massdriver.yaml resource type: %w", buildErr)
		}

		// Dereference the built schema
		opts := DereferenceOptions{
			Client: mdClient,
			Cwd:    filepath.Dir(path),
		}
		dereferencedAny, derefErr := DereferenceSchema(built, opts)
		if derefErr != nil {
			return nil, fmt.Errorf("failed to dereference resource type: %w", derefErr)
		}
		dereferenced, ok := dereferencedAny.(map[string]any)
		if !ok {
			return nil, errors.New("dereferenced resource type is not a map")
		}
		return dereferenced, nil
	}

	artdefBytes, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read resource type: %w", readErr)
	}

	var artdefMap map[string]any
	switch filepath.Ext(path) {
	case ".json":
		jsonErr := json.Unmarshal(artdefBytes, &artdefMap)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal resource type JSON: %w", jsonErr)
		}
	case ".yaml", ".yml":
		yamlErr := yaml.Unmarshal(artdefBytes, &artdefMap)
		if yamlErr != nil {
			return nil, fmt.Errorf("failed to unmarshal resource type YAML: %w", yamlErr)
		}
	default:
		return nil, fmt.Errorf("unsupported resource type file extension: %s", filepath.Ext(path))
	}

	// Dereferencing here. We may want to break this out in the future, but for now Reading and Dereferencing should be coupled.
	opts := DereferenceOptions{
		Client: mdClient,
		Cwd:    filepath.Dir(path),
	}
	dereferencedResourceTypeAny, derefErr := DereferenceSchema(artdefMap, opts)
	if derefErr != nil {
		return nil, fmt.Errorf("failed to dereference resource type: %w", derefErr)
	}

	dereferencedResourceType, ok := dereferencedResourceTypeAny.(map[string]any)
	if !ok {
		return nil, errors.New("dereferenced resource type is not a map")
	}

	return dereferencedResourceType, nil
}
