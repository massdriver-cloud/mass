package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestGetDeployment(t *testing.T) {
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployment": map[string]interface{}{
				"id":     "uuid1",
				"status": "PROVISIONING",
			},
		},
	})

	deployment, err := api.GetDeployment(client, "faux-org-id", "uuid1")

	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "PROVISIONING"

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}

func TestDeployPackage(t *testing.T) {
	want := "deployment-uuid1"
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployPackage": map[string]interface{}{
				"result": map[string]interface{}{
					"id": want,
				},
				"successful": true,
			},
		},
	})

	deployment, err := api.DeployPackage(client, "faux-org-id", "target-id", "manifest-id")

	if err != nil {
		t.Fatal(err)
	}

	got := deployment.ID

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
