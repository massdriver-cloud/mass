package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetBundle(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundle": map[string]any{
				"id":          "aws-aurora-postgres@1.2.3",
				"name":        "aws-aurora-postgres",
				"version":     "1.2.3",
				"description": "AWS Aurora PostgreSQL database",
				"icon":        "https://example.com/icon.png",
				"sourceUrl":   "https://github.com/example/bundle",
				"createdAt":   "2024-01-01T00:00:00Z",
				"updatedAt":   "2024-01-01T00:00:00Z",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	bundle, err := api.GetBundle(t.Context(), &mdClient, "aws-aurora-postgres@1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	if bundle.ID != "aws-aurora-postgres@1.2.3" {
		t.Errorf("got %s, wanted aws-aurora-postgres@1.2.3", bundle.ID)
	}
	if bundle.Name != "aws-aurora-postgres" {
		t.Errorf("got %s, wanted aws-aurora-postgres", bundle.Name)
	}
	if bundle.Version != "1.2.3" {
		t.Errorf("got %s, wanted 1.2.3", bundle.Version)
	}
}
