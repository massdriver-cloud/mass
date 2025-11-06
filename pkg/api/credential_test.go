package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestListCredentials(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"artifacts": map[string]any{
				"items": []map[string]any{
					{
						"id":        "uuid1",
						"name":      "artifact1",
						"type":      "massdriver/aws-iam-role",
						"updatedAt": "2025-01-01T00:00:00Z",
					},
					{
						"id":        "uuid2",
						"name":      "artifact2",
						"type":      "massdriver/gcp-service-account",
						"updatedAt": "2025-01-01T00:00:00Z",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
		Config: config.Config{
			OrganizationID: "org-123",
		},
	}

	credentials, err := api.ListCredentials(t.Context(), &mdClient)

	if err != nil {
		t.Fatal(err)
	}

	got := len(credentials)
	want := 2

	if got != want {
		t.Errorf("got %d credentials, wanted %d", got, want)
	}
}

func TestListArtifactsByType(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"artifacts": map[string]any{
				"items": []map[string]any{
					{
						"id":        "uuid1",
						"name":      "artifact1",
						"type":      "massdriver/aws-iam-role",
						"updatedAt": "2025-01-01T00:00:00Z",
					},
					{
						"id":        "uuid2",
						"name":      "artifact2",
						"type":      "massdriver/aws-iam-role",
						"updatedAt": "2025-01-01T00:00:00Z",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
		Config: config.Config{
			OrganizationID: "org-123",
		},
	}

	artifacts, err := api.ListArtifactsByType(t.Context(), &mdClient, "massdriver/aws-iam-role")

	if err != nil {
		t.Fatal(err)
	}

	got := len(artifacts)
	want := 2

	if got != want {
		t.Errorf("got %d artifacts, wanted %d", got, want)
	}
}
