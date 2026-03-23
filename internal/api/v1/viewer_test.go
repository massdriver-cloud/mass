package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetViewerAccount(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"__typename": "AccountViewer",
				"id":         "user-1",
				"email":      "user@example.com",
				"firstName":  "Jane",
				"lastName":   "Doe",
				"createdAt":  "2024-01-01T00:00:00Z",
				"updatedAt":  "2024-01-01T00:00:00Z",
				"defaultOrganization": map[string]any{
					"id":   "org-1",
					"name": "My Org",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	viewer, err := api.GetViewer(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	if viewer.Account == nil {
		t.Fatal("expected AccountViewer, got nil")
	}
	if viewer.ServiceAccount != nil {
		t.Error("expected ServiceAccount to be nil")
	}
	if viewer.Account.Email != "user@example.com" {
		t.Errorf("got %s, wanted user@example.com", viewer.Account.Email)
	}
	if viewer.Account.DefaultOrganization == nil || viewer.Account.DefaultOrganization.ID != "org-1" {
		t.Error("expected default organization with ID org-1")
	}
}

func TestGetViewerServiceAccount(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"__typename":  "ServiceAccountViewer",
				"id":          "sa-1",
				"name":        "ci-bot",
				"description": "CI/CD service account",
				"createdAt":   "2024-01-01T00:00:00Z",
				"updatedAt":   "2024-01-01T00:00:00Z",
				"organization": map[string]any{
					"id":   "org-1",
					"name": "My Org",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	viewer, err := api.GetViewer(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	if viewer.ServiceAccount == nil {
		t.Fatal("expected ServiceAccountViewer, got nil")
	}
	if viewer.Account != nil {
		t.Error("expected Account to be nil")
	}
	if viewer.ServiceAccount.Name != "ci-bot" {
		t.Errorf("got %s, wanted ci-bot", viewer.ServiceAccount.Name)
	}
	if viewer.ServiceAccount.Organization == nil || viewer.ServiceAccount.Organization.ID != "org-1" {
		t.Error("expected organization with ID org-1")
	}
}
