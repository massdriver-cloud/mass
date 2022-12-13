package api_test

import (
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestDeployPreviewEnvironment(t *testing.T) {
	prNumber := 69
	slug := fmt.Sprintf("p%d", prNumber)

	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployPreviewEnvironment": map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "envuuid1",
					"slug": slug,
				},
				"successful": true,
			},
		},
	})

	confMap := map[string]interface{}{
		"network": map[string]interface{}{
			"cidr": "10.0.0.0/16",
		},
		"cluster": map[string]interface{}{
			"maxNodes": 10,
		},
	}
	ciContext := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"title":  "First commit!",
			"number": prNumber,
		},
	}

	credentials := []api.Credential{}

	environment, err := api.DeployPreviewEnvironment(client, "faux-org-id", "faux-project-id", credentials, confMap, ciContext)

	if err != nil {
		t.Fatal(err)
	}

	got := environment.Slug
	want := "p69"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
