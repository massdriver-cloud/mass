// Package resourcetype provides CLI helpers around resource-type operations,
// thin wrappers over the Massdriver SDK's resource-type and OCI-repo services.
package resourcetype

import (
	"context"
	"encoding/json"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/resourcetypes"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// ResourceType is an alias of the SDK's resource-type record so consumers stay
// decoupled from the SDK import path.
type ResourceType = resourcetypes.ResourceType

// Get retrieves a resource type by name (optionally `name@version`) from
// Massdriver, including its resolved JSON schema.
func Get(ctx context.Context, mdClient *massdriver.Client, resourceTypeName string) (*ResourceType, error) {
	return mdClient.ResourceTypes.Get(ctx, resourceTypeName)
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

// List returns every resource type in the configured organization, sourced from
// the OCI repository catalog filtered to resource-type artifacts. The returned
// records carry only catalog metadata (ID, name, icon, timestamps); use [Get]
// to fetch a single resource type's schema.
func List(ctx context.Context, mdClient *massdriver.Client) ([]ResourceType, error) {
	seq := mdClient.OciRepos.Iter(ctx, ocirepos.ListInput{
		ArtifactType: ocirepos.ArtifactTypeResourceType,
	})
	repos, collectErr := types.Collect(seq)
	if collectErr != nil {
		return nil, collectErr
	}

	resourceTypes := make([]ResourceType, len(repos))
	for i, repo := range repos {
		resourceTypes[i] = ResourceType{
			ID:        repo.ID,
			Name:      repo.Name,
			Icon:      repo.Icon,
			CreatedAt: repo.CreatedAt,
			UpdatedAt: repo.UpdatedAt,
		}
	}
	return resourceTypes, nil
}
