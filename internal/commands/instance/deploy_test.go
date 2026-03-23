package instance_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunDeploy(t *testing.T) {
	responses := []any{
		gqlmock.MockQueryResponse("getPackage", api.Package{
			Manifest:    &api.Manifest{ID: "manifest-id"},
			Environment: &api.Environment{ID: "target-id"},
		}),
		gqlmock.MockMutationResponse("deployPackage", api.Deployment{
			ID:     "deployment-id",
			Status: "STARTED",
		}),
		gqlmock.MockQueryResponse("deployment", api.Deployment{
			ID:     "deployment-id",
			Status: "PENDING",
		}),
		gqlmock.MockQueryResponse("deployment", api.Deployment{
			ID:     "deployment-id",
			Status: "COMPLETED",
		}),
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithJSONResponseArray(responses),
	}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	deployment, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", "foo")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
