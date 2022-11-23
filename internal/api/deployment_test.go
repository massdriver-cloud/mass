package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
)

func TestGetDeployment(t *testing.T) {
	client := mockClientWithSingleJSONResponse(map[string]interface{}{
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
