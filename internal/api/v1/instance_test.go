package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetInstance(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"instance": map[string]any{
				"id":              "inst-uuid1",
				"name":            "my-database",
				"status":          "PROVISIONED",
				"version":         "~1.0",
				"releaseStrategy": "STABLE",
				"environment": map[string]any{
					"id":   "env-1",
					"name": "production",
				},
				"bundle": map[string]any{
					"id":      "aws-aurora-postgres@1.2.3",
					"name":    "aws-aurora-postgres",
					"version": "1.2.3",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	inst, err := api.GetInstance(t.Context(), &mdClient, "inst-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if inst.ID != "inst-uuid1" {
		t.Errorf("got %s, wanted inst-uuid1", inst.ID)
	}
	if inst.Status != "PROVISIONED" {
		t.Errorf("got %s, wanted PROVISIONED", inst.Status)
	}
	if inst.Bundle == nil || inst.Bundle.Name != "aws-aurora-postgres" {
		t.Error("expected bundle with name aws-aurora-postgres")
	}
}

func TestListInstances(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"instances": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":              "inst-1",
						"name":            "database",
						"status":          "PROVISIONED",
						"version":         "1.0.0",
						"releaseStrategy": "STABLE",
					},
					{
						"id":              "inst-2",
						"name":            "cache",
						"status":          "INITIALIZED",
						"version":         "~2.0",
						"releaseStrategy": "STABLE",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	instances, err := api.ListInstances(t.Context(), &mdClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 2 {
		t.Errorf("got %d instances, wanted 2", len(instances))
	}
}
