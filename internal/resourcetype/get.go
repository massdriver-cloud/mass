// Package resourcetype provides CLI helpers around resource-type operations.
//
// The underlying GraphQL surface lives in [github.com/massdriver-cloud/mass/internal/api],
// a temporary holding pen for ops not yet exposed by the Massdriver SDK. When
// the SDK adds native resource-type support this package collapses to thin
// wrappers over the SDK and `internal/api` is deleted.
package resourcetype

import (
	"context"
	"encoding/json"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// ResourceType is an alias of [api.ResourceType] so consumers stay decoupled
// from the holding-pen package import.
type ResourceType = api.ResourceType

// Get retrieves a resource type by name from the Massdriver API.
func Get(ctx context.Context, mdClient *massdriver.Client, resourceTypeName string) (*ResourceType, error) {
	return api.GetResourceType(ctx, mdClient, resourceTypeName)
}

// GetAsMap retrieves a resource type and returns it as a generic map.
func GetAsMap(ctx context.Context, mdClient *massdriver.Client, resourceTypeName string) (map[string]any, error) {
	rt, getErr := Get(ctx, mdClient, resourceTypeName)
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

// List returns every resource type in the configured organization.
func List(ctx context.Context, mdClient *massdriver.Client) ([]ResourceType, error) {
	return api.ListResourceTypes(ctx, mdClient)
}
