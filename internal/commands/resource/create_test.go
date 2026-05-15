package resource_test

import (
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/resource"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/resources"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// fakeResourceAPI is a hand-rolled stub for resource.API. Tests set only the
// fields the case under test exercises.
type fakeResourceAPI struct {
	resource      *types.Resource
	resourceType  *resourcetype.ResourceType
	resourceTypes []resourcetype.ResourceType

	getResourceErr      error
	getResourceTypeErr  error
	listResourceTypeErr error
	createErr           error
	updateErr           error
	deleteErr           error

	gotCreateInput  resources.CreateInput
	gotCreateTypeID string
	gotUpdateInput  resources.UpdateInput
	gotUpdateID     string
	gotDeleteID     string
}

func (f *fakeResourceAPI) GetResource(_ context.Context, _ string) (*types.Resource, error) {
	return f.resource, f.getResourceErr
}

func (f *fakeResourceAPI) CreateResource(_ context.Context, resourceTypeID string, in resources.CreateInput) (*types.Resource, error) {
	f.gotCreateTypeID = resourceTypeID
	f.gotCreateInput = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.resource, nil
}

func (f *fakeResourceAPI) UpdateResource(_ context.Context, id string, in resources.UpdateInput) (*types.Resource, error) {
	f.gotUpdateID = id
	f.gotUpdateInput = in
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.resource, nil
}

func (f *fakeResourceAPI) DeleteResource(_ context.Context, id string) (*types.Resource, error) {
	f.gotDeleteID = id
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return f.resource, nil
}

func (f *fakeResourceAPI) GetResourceType(_ context.Context, _ string) (*resourcetype.ResourceType, error) {
	return f.resourceType, f.getResourceTypeErr
}

func (f *fakeResourceAPI) ListResourceTypes(_ context.Context) ([]resourcetype.ResourceType, error) {
	return f.resourceTypes, f.listResourceTypeErr
}

func TestResourceImport(t *testing.T) {
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "resource-id", Name: "resource-name"},
		resourceType: &resourcetype.ResourceType{
			ID:   "massdriver/fake-resource-schema",
			Name: "massdriver/fake-resource-schema",
			Schema: map[string]any{
				"$id":     "id",
				"$schema": "http://json-schema.org/draft-07/schema",
				"type":    "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
	}

	got, err := resource.RunCreate(t.Context(), api, "resource-name", "massdriver/fake-resource-schema", "testdata/resource.json")
	if err != nil {
		t.Fatal(err)
	}

	if got != "resource-id" {
		t.Errorf("got %s, wanted resource-id", got)
	}
	if api.gotCreateTypeID != "massdriver/fake-resource-schema" {
		t.Errorf("got resource type id %q, wanted massdriver/fake-resource-schema", api.gotCreateTypeID)
	}
	if api.gotCreateInput.Name != "resource-name" {
		t.Errorf("got name %q, wanted resource-name", api.gotCreateInput.Name)
	}
}
