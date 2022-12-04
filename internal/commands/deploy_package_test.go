package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestDeployPackage(t *testing.T) {
	responses := []interface{}{
		mockQueryResponse("getPackageByNamingConvention", api.Package{
			Manifest: api.Manifest{ID: "manifest-id"},
			Target:   api.Target{ID: "target-id"},
		}),
		mockMutationResponse("deployPackage", api.Deployment{
			ID:     "deployment-id",
			Status: "STARTED",
		}),
		mockQueryResponse("deployment", api.Deployment{
			ID:     "deployment-id",
			Status: "PENDING",
		}),
		mockQueryResponse("deployment", api.Deployment{
			ID:     "deployment-id",
			Status: "COMPLETED",
		}),
	}

	client := mockClientWithJSONResponseArray(responses)
	commands.DeploymentStatusSleep = 0

	deployment, err := commands.DeployPackage(client, "faux-org-id", "ecomm-prod-cache")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
