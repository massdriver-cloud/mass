package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
)

func TestDeployPackage(t *testing.T) {
	responses := []interface{}{
		gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
			Manifest: api.Manifest{ID: "manifest-id"},
			Target:   api.Target{ID: "target-id"},
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

	client := gqlmock.NewClientWithJSONResponseArray(responses)
	commands.DeploymentStatusSleep = 0

	deployment, err := commands.DeployPackage(client, "faux-org-id", "ecomm-prod-cache", "foo")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
