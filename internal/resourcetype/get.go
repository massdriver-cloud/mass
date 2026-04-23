package resourcetype

import (
	"context"
	"encoding/json"

	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Get retrieves a resource type by name from the Massdriver API.
func Get(ctx context.Context, mdClient *client.Client, definitionName string) (*api.ResourceType, error) {
	return api.GetResourceType(ctx, mdClient, definitionName)
}

// GetAsMap retrieves a resource type and returns it as a generic map.
func GetAsMap(ctx context.Context, mdClient *client.Client, definitionName string) (map[string]any, error) {
	rt, getErr := Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return nil, getErr
	}

	rtData, marshallErr := json.Marshal(rt)
	if marshallErr != nil {
		return nil, marshallErr
	}

	var result map[string]any
	unmarshalErr := json.Unmarshal(rtData, &result)
	return result, unmarshalErr
}
