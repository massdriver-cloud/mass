package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetDeployment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployment": map[string]any{
				"id":                 "dep-uuid1",
				"status":             "COMPLETED",
				"action":             "PROVISION",
				"version":            "1.2.3",
				"message":            "Deployed successfully",
				"createdAt":          "2024-01-01T00:00:00Z",
				"updatedAt":          "2024-01-01T00:05:00Z",
				"lastTransitionedAt": "2024-01-01T00:05:00Z",
				"elapsedTime":        300,
				"deployedBy":         "user@example.com",
				"instance": map[string]any{
					"id":   "inst-1",
					"name": "my-database",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployment, err := api.GetDeployment(t.Context(), &mdClient, "dep-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if deployment.ID != "dep-uuid1" {
		t.Errorf("got %s, wanted dep-uuid1", deployment.ID)
	}
	if deployment.Status != "COMPLETED" {
		t.Errorf("got %s, wanted COMPLETED", deployment.Status)
	}
	if deployment.Action != "PROVISION" {
		t.Errorf("got %s, wanted PROVISION", deployment.Action)
	}
	if deployment.ElapsedTime != 300 {
		t.Errorf("got %d, wanted 300", deployment.ElapsedTime)
	}
}

func TestListDeployments(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployments": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":          "dep-1",
						"status":      "COMPLETED",
						"action":      "PROVISION",
						"version":     "1.0.0",
						"createdAt":   "2024-01-01T00:00:00Z",
						"updatedAt":   "2024-01-01T00:05:00Z",
						"elapsedTime": 300,
					},
					{
						"id":          "dep-2",
						"status":      "RUNNING",
						"action":      "DECOMMISSION",
						"version":     "1.0.0",
						"createdAt":   "2024-01-02T00:00:00Z",
						"updatedAt":   "2024-01-02T00:01:00Z",
						"elapsedTime": 60,
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployments, err := api.ListDeployments(t.Context(), &mdClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployments) != 2 {
		t.Errorf("got %d deployments, wanted 2", len(deployments))
	}
}
