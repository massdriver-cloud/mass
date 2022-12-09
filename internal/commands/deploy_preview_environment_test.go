package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestDeployPreviewEnvironment(t *testing.T) {
	projectSlug := "ecomm"
	envSlug := "p9000"
	responses := []interface{}{
		mockMutationResponse("deployPreviewEnvironment", map[string]interface{}{
			"id":   "envUUID",
			"slug": envSlug,
			"project": map[string]interface{}{
				"id":   "projUUID",
				"slug": projectSlug,
			},
		}),
	}
	client := mockClientWithJSONResponseArray(responses)

	env, err := commands.DeployPreviewEnvironment(client, "faux-org-id", projectSlug)

	if err != nil {
		t.Fatal(err)
	}

	got := env.URL
	want := "https://app.massdriver.cloud/projects/projUUID/targets/envUUID"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
