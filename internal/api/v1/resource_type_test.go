package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
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
	mdClient := client.Client{GQLv1: gqlClient}

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
	mdClient := client.Client{GQLv1: gqlClient}

	rts, err := api.ListResourceTypes(t.Context(), &mdClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(rts) != 2 {
		t.Errorf("got %d resource types, wanted 2", len(rts))
	}
}

func TestListResourceTypesWithFilter(t *testing.T) {
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
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	filter := api.ResourceTypesFilter{
		Id: &api.StringFilter{Eq: "aws-iam-role"},
	}
	rts, err := api.ListResourceTypes(t.Context(), &mdClient, &filter)
	if err != nil {
		t.Fatal(err)
	}

	if len(rts) != 1 {
		t.Errorf("got %d resource types, wanted 1", len(rts))
	}
}
