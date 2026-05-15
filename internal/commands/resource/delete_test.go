package resource_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/resource"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestResourceDeleteForce(t *testing.T) {
	// --force skips the prompt entirely.
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "rid-1", Name: "doomed"},
	}

	if err := resource.RunDelete(t.Context(), api, "rid-1", true, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	if api.gotDeleteID != "rid-1" {
		t.Errorf("got delete id %q, wanted rid-1", api.gotDeleteID)
	}
}

func TestResourceDeleteConfirmed(t *testing.T) {
	// User types the resource name → delete proceeds.
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "rid-1", Name: "doomed"},
	}

	if err := resource.RunDelete(t.Context(), api, "rid-1", false, strings.NewReader("doomed\n")); err != nil {
		t.Fatal(err)
	}
	if api.gotDeleteID != "rid-1" {
		t.Errorf("got delete id %q, wanted rid-1", api.gotDeleteID)
	}
}

func TestResourceDeleteCancelled(t *testing.T) {
	// Typing anything other than the resource name aborts.
	api := &fakeResourceAPI{
		resource: &types.Resource{ID: "rid-1", Name: "doomed"},
	}

	if err := resource.RunDelete(t.Context(), api, "rid-1", false, strings.NewReader("nope\n")); err != nil {
		t.Fatal(err)
	}
	if api.gotDeleteID != "" {
		t.Errorf("expected no delete call, but got id %q", api.gotDeleteID)
	}
}
