package definition

import (
	"context"
	"encoding/json"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Get retrieves an artifact definition by name from the Massdriver API.
func Get(ctx context.Context, mdClient *client.Client, definitionName string) (*api.ArtifactDefinitionWithSchema, error) {
	return api.GetArtifactDefinition(ctx, mdClient, definitionName)
}

// GetAsMap retrieves an artifact definition and returns it as a generic map.
func GetAsMap(ctx context.Context, mdClient *client.Client, definitionName string) (map[string]any, error) {
	ad, getErr := Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return nil, getErr
	}

	adData, marshallErr := json.Marshal(ad)
	if marshallErr != nil {
		return nil, marshallErr
	}

	var result map[string]any
	unmarshalErr := json.Unmarshal(adData, &result)
	return result, unmarshalErr
}
