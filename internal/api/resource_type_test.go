package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetResourceType(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"resourceType": map[string]any{
				"id":                    "aws-iam-role",
				"name":                  "AWS IAM Role",
				"icon":                  "https://example.com/iam.png",
				"connectionOrientation": "LINK",
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	rt, err := api.GetResourceType(t.Context(), &mdClient, "aws-iam-role")
	if err != nil {
		t.Fatal(err)
	}

	if rt.ID != "aws-iam-role" {
		t.Errorf("got ID %s, wanted aws-iam-role", rt.ID)
	}
	if rt.Name != "AWS IAM Role" {
		t.Errorf("got name %s, wanted AWS IAM Role", rt.Name)
	}
	if rt.Icon != "https://example.com/iam.png" {
		t.Errorf("got icon %s, wanted https://example.com/iam.png", rt.Icon)
	}
	if rt.ConnectionOrientation != "LINK" {
		t.Errorf("got connectionOrientation %s, wanted LINK", rt.ConnectionOrientation)
	}
}

func TestListResourceTypes(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"resourceTypes": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":                    "aws-iam-role",
						"name":                  "AWS IAM Role",
						"connectionOrientation": "LINK",
					},
					{
						"id":                    "kubernetes-cluster",
						"name":                  "Kubernetes Cluster",
						"connectionOrientation": "ENVIRONMENT_DEFAULT",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	rts, err := api.ListResourceTypes(t.Context(), &mdClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(rts) != 2 {
		t.Errorf("got %d resource types, wanted 2", len(rts))
	}
}

func TestPublishResourceType(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"publishResourceType": map[string]any{
				"result": map[string]any{
					"id":                    "aws-iam-role",
					"name":                  "AWS IAM Role",
					"connectionOrientation": "LINK",
					"schema": map[string]any{
						"$md":  map[string]any{"name": "aws-iam-role"},
						"type": "object",
					},
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	rt, err := api.PublishResourceType(t.Context(), &mdClient, api.PublishResourceTypeInput{
		Schema: map[string]any{
			"$md":  map[string]any{"name": "aws-iam-role"},
			"type": "object",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if rt.ID != "aws-iam-role" {
		t.Errorf("got ID %s, wanted aws-iam-role", rt.ID)
	}
}

func TestPublishResourceTypeFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"publishResourceType": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{
						"code":    "validation",
						"field":   "schema",
						"message": "schema is missing $md.name",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	_, err := api.PublishResourceType(t.Context(), &mdClient, api.PublishResourceTypeInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteResourceType(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deleteResourceType": map[string]any{
				"result": map[string]any{
					"id":   "aws-iam-role",
					"name": "AWS IAM Role",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	rt, err := api.DeleteResourceType(t.Context(), &mdClient, "aws-iam-role")
	if err != nil {
		t.Fatal(err)
	}

	if rt.ID != "aws-iam-role" {
		t.Errorf("got ID %s, wanted aws-iam-role", rt.ID)
	}
}

func TestDeleteResourceTypeFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deleteResourceType": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{
						"code":    "conflict",
						"field":   "id",
						"message": "resource type is still in use",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	_, err := api.DeleteResourceType(t.Context(), &mdClient, "aws-iam-role")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
