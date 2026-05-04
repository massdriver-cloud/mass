package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
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
				"description": "Aurora PostgreSQL cluster",
				"icon":        "https://example.com/icon.png",
				"sourceUrl":   "https://github.com/example/repo",
				"repo":        "aws-aurora-postgres",
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	bundle, err := api.GetBundle(t.Context(), &mdClient, "aws-aurora-postgres@1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	if bundle.ID != "aws-aurora-postgres@1.2.3" {
		t.Errorf("got ID %s, wanted aws-aurora-postgres@1.2.3", bundle.ID)
	}
	if bundle.Name != "aws-aurora-postgres" {
		t.Errorf("got name %s, wanted aws-aurora-postgres", bundle.Name)
	}
	if bundle.Version != "1.2.3" {
		t.Errorf("got version %s, wanted 1.2.3", bundle.Version)
	}
	if bundle.Repo != "aws-aurora-postgres" {
		t.Errorf("got repo %s, wanted aws-aurora-postgres", bundle.Repo)
	}
}

func TestListBundles(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundles": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":      "aws-aurora-postgres@1.2.3",
						"name":    "aws-aurora-postgres",
						"version": "1.2.3",
						"repo":    "aws-aurora-postgres",
					},
					{
						"id":      "aws-s3@2.0.0",
						"name":    "aws-s3",
						"version": "2.0.0",
						"repo":    "aws-s3",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	bundles, err := api.ListBundles(t.Context(), &mdClient, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(bundles) != 2 {
		t.Errorf("got %d bundles, wanted 2", len(bundles))
	}
}

func TestListBundlesWithFilter(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundles": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":      "aws-aurora-postgres@1.2.3",
						"name":    "aws-aurora-postgres",
						"version": "1.2.3",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv2: gqlClient}

	filter := api.BundlesFilter{
		OciRepo: &api.OciRepoNameFilter{Eq: "aws-aurora-postgres"},
	}
	bundles, err := api.ListBundles(t.Context(), &mdClient, &filter, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(bundles) != 1 {
		t.Errorf("got %d bundles, wanted 1", len(bundles))
	}
}
