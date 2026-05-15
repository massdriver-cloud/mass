package resource_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/resource"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestResourceUpdate(t *testing.T) {
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "resource-id", Name: "resource-name"},
	}

	got, err := resource.RunUpdate(t.Context(), api, "resource-id", "resource-name", "testdata/resource.json")
	if err != nil {
		t.Fatal(err)
	}

	if got != "resource-id" {
		t.Errorf("got %s, wanted resource-id", got)
	}
	if api.gotUpdateInput.Name != "resource-name" {
		t.Errorf("got name %q, wanted resource-name", api.gotUpdateInput.Name)
	}
}

func TestResourceUpdateWithoutName(t *testing.T) {
	// When no name is provided, RunUpdate fetches the existing resource first.
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "resource-id", Name: "existing-name"},
	}

	got, err := resource.RunUpdate(t.Context(), api, "resource-id", "", "testdata/resource.json")
	if err != nil {
		t.Fatal(err)
	}

	if got != "resource-id" {
		t.Errorf("got %s, wanted resource-id", got)
	}
	if api.gotUpdateInput.Name != "existing-name" {
		t.Errorf("got name %q, wanted existing-name", api.gotUpdateInput.Name)
	}
}
