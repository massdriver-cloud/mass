package pkg_test

import (
	"net/http"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/pkg"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunReset(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackage", api.Package{
				ID:          "pkg-uuid1",
				Slug:        "ecomm-prod-cache",
				Manifest:    &api.Manifest{ID: "manifest-id"},
				Environment: &api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			return gqlmock.MockMutationResponse("resetPackage", map[string]any{
				"id":     "pkg-uuid1",
				"slug":   "ecomm-prod-cache",
				"status": "ready",
			})
		},
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}

	pkg, err := pkg.RunReset(t.Context(), &mdClient, "ecomm-prod-cache")
	if err != nil {
		t.Fatal(err)
	}

	if pkg.ID != "pkg-uuid1" {
		t.Errorf("got %v, wanted %v", pkg.ID, "pkg-uuid1")
	}
	if pkg.Slug != "ecomm-prod-cache" {
		t.Errorf("got %v, wanted %v", pkg.Slug, "ecomm-prod-cache")
	}
	if pkg.Status != "ready" {
		t.Errorf("got %v, wanted %v", pkg.Status, "ready")
	}
}
