package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetDeployment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployment": map[string]any{
				"id":      "dep-uuid1",
				"status":  "COMPLETED",
				"action":  "PROVISION",
				"version": "1.2.3",
				"instance": map[string]any{
					"id":   "inst-1",
					"name": "db",
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	dep, err := api.GetDeployment(t.Context(), &mdClient, "dep-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if dep.ID != "dep-uuid1" {
		t.Errorf("got %s, wanted dep-uuid1", dep.ID)
	}
	if dep.Status != "COMPLETED" {
		t.Errorf("got %s, wanted COMPLETED", dep.Status)
	}
	if dep.Instance == nil || dep.Instance.ID != "inst-1" {
		t.Errorf("expected instance inst-1")
	}
}

func TestListDeployments(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deployments": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{"id": "dep-1", "status": "COMPLETED", "action": "PROVISION"},
					{"id": "dep-2", "status": "RUNNING", "action": "PROVISION"},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	deployments, err := api.ListDeployments(t.Context(), &mdClient, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployments) != 2 {
		t.Errorf("got %d deployments, wanted 2", len(deployments))
	}
}

func TestCreateDeployment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createDeployment": map[string]any{
				"result": map[string]any{
					"id":      "dep-new",
					"status":  "PENDING",
					"action":  "PROVISION",
					"version": "1.2.3",
					"message": "Initial deployment",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	dep, err := api.CreateDeployment(t.Context(), &mdClient, "inst-1", api.CreateDeploymentInput{
		Action:  api.DeploymentActionProvision,
		Message: "Initial deployment",
		Params:  map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if dep.ID != "dep-new" {
		t.Errorf("got %s, wanted dep-new", dep.ID)
	}
}

func TestCreateDeploymentFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createDeployment": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{
						"code":    "invalid",
						"field":   "params",
						"message": "params failed validation",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	_, err := api.CreateDeployment(t.Context(), &mdClient, "inst-1", api.CreateDeploymentInput{
		Action: api.DeploymentActionProvision,
		Params: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
