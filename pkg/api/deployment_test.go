package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestGetDeployment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployment": map[string]any{
				"id":     "uuid1",
				"status": "PROVISIONING",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployment, err := api.GetDeployment(t.Context(), &mdClient, "uuid1")

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
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployPackage": map[string]any{
				"result": map[string]any{
					"id": want,
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployment, err := api.DeployPackage(t.Context(), &mdClient, "target-id", "manifest-id", "foo")

	if err != nil {
		t.Fatal(err)
	}

	got := deployment.ID

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestGetDeploymentLogs(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deploymentLogStream": map[string]any{
				"id": "log-stream-1",
				"logs": []map[string]any{
					{
						"content": "Starting deployment...\n",
						"metadata": map[string]any{
							"step":      "provision",
							"timestamp": "2024-01-01T00:00:00Z",
							"index":     0,
						},
					},
					{
						"content": "Deployment completed successfully\n",
						"metadata": map[string]any{
							"step":      "complete",
							"timestamp": "2024-01-01T00:05:00Z",
							"index":     1,
						},
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
		Config: config.Config{
			OrganizationID: "org-1",
		},
	}

	logs, err := api.GetDeploymentLogs(t.Context(), &mdClient, "deployment-1")

	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 2 {
		t.Fatalf("got %d logs, wanted 2", len(logs))
	}

	if logs[0].Content != "Starting deployment...\n" {
		t.Errorf("got %s, wanted 'Starting deployment...\\n'", logs[0].Content)
	}

	if logs[0].Step != "provision" {
		t.Errorf("got %s, wanted 'provision'", logs[0].Step)
	}

	if logs[1].Content != "Deployment completed successfully\n" {
		t.Errorf("got %s, wanted 'Deployment completed successfully\\n'", logs[1].Content)
	}

	if logs[1].Step != "complete" {
		t.Errorf("got %s, wanted 'complete'", logs[1].Step)
	}
}
