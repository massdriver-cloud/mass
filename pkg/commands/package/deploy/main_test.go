package deploy_test

import (
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/package/deploy"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestDeployPackage(t *testing.T) {
	responses := []interface{}{
		gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
			Manifest:    api.Manifest{ID: "manifest-id"},
			Environment: api.Environment{ID: "target-id"},
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
	deploy.DeploymentStatusSleep = 0

	deployment, err := deploy.Run(context.Background(), &mdClient, "ecomm-prod-cache", "foo")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
