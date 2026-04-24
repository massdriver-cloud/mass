package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestListLinks(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"project": map[string]any{
				"blueprint": map[string]any{
					"links": map[string]any{
						"cursor": map[string]any{},
						"items": []map[string]any{
							{
								"id":            "link-1",
								"fromField":     "authentication",
								"toField":       "database",
								"fromComponent": map[string]any{"id": "ecomm-db"},
								"toComponent":   map[string]any{"id": "ecomm-app"},
							},
						},
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	filter := &api.LinksFilter{
		FromComponentId: &api.IdFilter{Eq: "ecomm-db"},
		ToComponentId:   &api.IdFilter{Eq: "ecomm-app"},
	}
	links, err := api.ListLinks(t.Context(), &mdClient, "ecomm", filter)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Fatalf("got %d links, wanted 1", len(links))
	}
	if links[0].FromField != "authentication" || links[0].ToField != "database" {
		t.Errorf("got %s→%s, wanted authentication→database", links[0].FromField, links[0].ToField)
	}
}

func TestAddComponent(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"addComponent": map[string]any{
				"result": map[string]any{
					"id":   "ecomm-db",
					"name": "Primary Database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	comp, err := api.AddComponent(t.Context(), &mdClient, "ecomm", "aws-rds-cluster", api.AddComponentInput{
		Id:   "db",
		Name: "Primary Database",
	})
	if err != nil {
		t.Fatal(err)
	}

	if comp.ID != "ecomm-db" {
		t.Errorf("got ID %s, wanted ecomm-db", comp.ID)
	}
}

func TestAddComponentFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"addComponent": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{"code": "validation", "field": "id", "message": "id is required"},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	_, err := api.AddComponent(t.Context(), &mdClient, "ecomm", "aws-rds-cluster", api.AddComponentInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveComponent(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"removeComponent": map[string]any{
				"result":     map[string]any{"id": "ecomm-db", "name": "db"},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	comp, err := api.RemoveComponent(t.Context(), &mdClient, "ecomm-db")
	if err != nil {
		t.Fatal(err)
	}
	if comp.ID != "ecomm-db" {
		t.Errorf("got %s, wanted ecomm-db", comp.ID)
	}
}

func TestLinkComponents(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"linkComponents": map[string]any{
				"result": map[string]any{
					"id":        "link-new",
					"fromField": "authentication",
					"toField":   "database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	link, err := api.LinkComponents(t.Context(), &mdClient, api.LinkComponentsInput{
		FromComponentId: "ecomm-db",
		FromField:       "authentication",
		FromVersion:     "~1.0",
		ToComponentId:   "ecomm-app",
		ToField:         "database",
		ToVersion:       "~2.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	if link.ID != "link-new" {
		t.Errorf("got %s, wanted link-new", link.ID)
	}
}

func TestUnlinkComponents(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"unlinkComponents": map[string]any{
				"result": map[string]any{
					"id":        "link-1",
					"fromField": "authentication",
					"toField":   "database",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	link, err := api.UnlinkComponents(t.Context(), &mdClient, "link-1")
	if err != nil {
		t.Fatal(err)
	}
	if link.ID != "link-1" {
		t.Errorf("got %s, wanted link-1", link.ID)
	}
}
