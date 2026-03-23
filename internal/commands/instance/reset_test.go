package instance_test

import (
	"net/http"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunReset(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackage", api.Package{
				ID:          "instance-uuid1",
				Slug:        "ecomm-prod-cache",
				Manifest:    &api.Manifest{ID: "manifest-id"},
				Environment: &api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			return gqlmock.MockMutationResponse("resetPackage", map[string]any{
				"id":     "instance-uuid1",
				"slug":   "ecomm-prod-cache",
				"status": "ready",
			})
		},
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}

	instance, err := instance.RunReset(t.Context(), &mdClient, "ecomm-prod-cache")
	if err != nil {
		t.Fatal(err)
	}

	if instance.ID != "instance-uuid1" {
		t.Errorf("got %v, wanted %v", instance.ID, "instance-uuid1")
	}
	if instance.Slug != "ecomm-prod-cache" {
		t.Errorf("got %v, wanted %v", instance.Slug, "ecomm-prod-cache")
	}
	if instance.Status != "ready" {
		t.Errorf("got %v, wanted %v", instance.Status, "ready")
	}
}
