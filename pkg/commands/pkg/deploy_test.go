package pkg_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/pkg"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunDeploy(t *testing.T) {
	responses := []any{
		gqlmock.QueryResponse{
			Data: map[string]any{
				"getPackage": map[string]any{
					"package": map[string]any{
						"slug": "ecomm-prod-cache",
						"manifest": map[string]any{
							"id": "manifest-id",
						},
						"environment": map[string]any{
							"id": "target-id",
						},
					},
				},
			},
		},
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
	pkg.DeploymentStatusSleep = 0

	deployment, err := pkg.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", "foo")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
