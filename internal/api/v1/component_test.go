package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestAddComponent(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"addComponent": map[string]any{
				"result": map[string]any{
					"id":          "database",
					"name":        "Billing Database",
					"description": "Main billing database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	comp, err := api.AddComponent(t.Context(), &mdClient, "proj-1", api.AddComponentInput{
		Id:          "database",
		Name:        "Billing Database",
		BundleName:  "aws-aurora-postgres",
		Description: "Main billing database",
	})
	if err != nil {
		t.Fatal(err)
	}

	if comp.ID != "database" {
		t.Errorf("got %s, wanted database", comp.ID)
	}
	if comp.Name != "Billing Database" {
		t.Errorf("got %s, wanted Billing Database", comp.Name)
	}
}

func TestRemoveComponent(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"removeComponent": map[string]any{
				"result": map[string]any{
					"id":   "database",
					"name": "Billing Database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	comp, err := api.RemoveComponent(t.Context(), &mdClient, "proj-1", "database")
	if err != nil {
		t.Fatal(err)
	}

	if comp.ID != "database" {
		t.Errorf("got %s, wanted database", comp.ID)
	}
}

func TestLinkComponents(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"linkComponents": map[string]any{
				"result": map[string]any{
					"id":        "link-1",
					"fromField": "postgres",
					"toField":   "database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	link, err := api.LinkComponents(t.Context(), &mdClient, "proj-1", api.LinkComponentsInput{
		From:        "database",
		FromField:   "postgres",
		FromVersion: "~1.0",
		To:          "app",
		ToField:     "database",
		ToVersion:   "~2.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	if link.ID != "link-1" {
		t.Errorf("got %s, wanted link-1", link.ID)
	}
	if link.FromField != "postgres" {
		t.Errorf("got %s, wanted postgres", link.FromField)
	}
}

func TestUnlinkComponents(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"unlinkComponents": map[string]any{
				"result": map[string]any{
					"id": "link-1",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	link, err := api.UnlinkComponents(t.Context(), &mdClient, "link-1")
	if err != nil {
		t.Fatal(err)
	}

	if link.ID != "link-1" {
		t.Errorf("got %s, wanted link-1", link.ID)
	}
}
