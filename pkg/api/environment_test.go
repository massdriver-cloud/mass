package api_test

import (
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
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

	credentials := []api.Credential{}

	packageParams := map[string]api.PreviewPackage{
		"network": {
			Params: map[string]interface{}{
				"cidr": "10.0.0.0/16",
			},
		},

		"cluster": {
			Params: map[string]interface{}{
				"maxNodes": 10,
			},
		},
	}

	ciContext := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"title":  "First commit!",
			"number": prNumber,
		},
	}

	environment, err := api.DeployPreviewEnvironment(client, "faux-org-id", "faux-project-id", credentials, packageParams, ciContext)

	if err != nil {
		t.Fatal(err)
	}

	got := environment.Slug
	want := "p69"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestDecommissionPreviewEnvironment(t *testing.T) {
	prNumber := 69
	targetSlug := fmt.Sprintf("p%d", prNumber)
	projectTargetSlug := "ecomm-" + targetSlug
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"decommissionPreviewEnvironment": map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "envuuid1",
					"slug": targetSlug,
				},
				"successful": true,
			},
		},
	})

	environment, err := api.DecommissionPreviewEnvironment(client, "faux-org-id", projectTargetSlug)

	if err != nil {
		t.Fatal(err)
	}

	got := environment.Slug
	want := "p69"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestDeployPreviewEnvironmentFailsWithBothParamsAndRemoteRefs(t *testing.T) {
	prNumber := 69
	slug := fmt.Sprintf("p%d", prNumber)

	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployPreviewEnvironment": map[string]interface{}{
				"result": map[string]interface{}{
					"slug": slug,
					"id":   "envuuid1",
				},
				"successful": true,
			},
		},
	})

	credentials := []api.Credential{}

	packageParams := map[string]api.PreviewPackage{
		"network": {
			Params: map[string]interface{}{
				"cidr": "10.0.0.0/16",
			},
			RemoteReferences: []api.RemoteRef{
				{
					ArtifactID: "00000000-0000-0000-0000-000000000000",
					Field:      "some-field",
				},
			},
		},
		"cluster": {
			Params: map[string]interface{}{
				"maxNodes": 9,
			},
		},
	}

	ciContext := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"title":  "First commit!",
			"number": prNumber,
		},
	}

	_, err := api.DeployPreviewEnvironment(client, "faux-org-id", "faux-project-id", credentials, packageParams, ciContext)

	if err == nil {
		t.Error("expected error when   both params and remote references are set, got nil")
	}

	expectedError := "package 'network': \"params\" and \"remoteReferences\" are mutually exclusive"
	if err.Error() != expectedError {
		t.Errorf("got error %q, wanted %q", err.Error(), expectedError)
	}
}
