package definition

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"gopkg.in/yaml.v3"
)

func Read(ctx context.Context, mdClient *client.Client, path string) (map[string]any, error) {
	artdefBytes, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read artifact definition: %w", readErr)
	}

	var artdefMap map[string]any
	switch filepath.Ext(path) {
	case ".json":
		jsonErr := json.Unmarshal(artdefBytes, &artdefMap)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal artifact definition JSON: %w", jsonErr)
		}
	case ".yaml", ".yml":
		yamlErr := yaml.Unmarshal(artdefBytes, &artdefMap)
		if yamlErr != nil {
			return nil, fmt.Errorf("failed to unmarshal artifact definition YAML: %w", yamlErr)
		}
	default:
		return nil, fmt.Errorf("unsupported artifact definition file extension: %s", filepath.Ext(path))
	}

	// Dereferencing here. We may want to break this out in the future, but for now Reading and Dereferencing should be coupled.
	opts := DereferenceOptions{
		Client: mdClient,
		Cwd:    filepath.Dir(path),
	}
	dereferencedArtifactAny, derefErr := DereferenceSchema(artdefMap, opts)
	if derefErr != nil {
		return nil, fmt.Errorf("failed to dereference artifact definition: %w", derefErr)
	}

	dereferencedArtifact, ok := dereferencedArtifactAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("dereferenced artifact definition is not a map")
	}

	return dereferencedArtifact, nil
}
