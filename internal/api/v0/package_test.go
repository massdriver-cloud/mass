package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPackage(t *testing.T) {
	pkgName := "ecomm-prod-cache"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"package": map[string]any{
				"slug": pkgName,
				"bundle": map[string]any{
					"id": "bundle-id",
				},
				"manifest": map[string]any{
					"id": "manifest-id",
				},
				"environment": map[string]any{
					"id": "target-id",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetPackage(t.Context(), &mdClient, pkgName)

	if err != nil {
		t.Fatal(err)
	}

	want := &api.Package{
		Slug: "ecomm-prod-cache",
		Bundle: &api.Bundle{
			ID: "bundle-id",
		},
		Manifest: &api.Manifest{
			ID: "manifest-id",
		},
		Environment: &api.Environment{
			ID:      "target-id",
			Project: &api.Project{},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestConfigurePackage(t *testing.T) {
	params := map[string]any{
		"cidr": "10.0.0.0/16",
	}

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"configurePackage": map[string]any{
				"result": map[string]any{
					"id":     "pkg-uuid1",
					"params": params,
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	pkg, err := api.ConfigurePackage(t.Context(), &mdClient, "faux-pkg-id", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestResetPackage(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"resetPackage": map[string]any{
				"result": map[string]any{
					"id":     "pkg-uuid1",
					"slug":   "ecomm-prod-cache",
					"status": "ready",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	pkg, err := api.ResetPackage(t.Context(), &mdClient, "pkg-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if pkg.ID != "pkg-uuid1" {
		t.Errorf("got %v, wanted %v", pkg.ID, "pkg-uuid1")
	}
	if pkg.Slug != "ecomm-prod-cache" {
		t.Errorf("got %v, wanted %v", pkg.Slug, "ecomm-prod-cache")
	}
	if pkg.Status != "ready" {
		t.Errorf("got %v, wanted %v", pkg.Status, "ready")
	}
}

func TestGetPackage_NilDeployments(t *testing.T) {
	pkgName := "ecomm-prod-cache"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"package": map[string]any{
				"slug":             pkgName,
				"status":           "provisioned",
				"deployedVersion":  nil,
				"latestDeployment": nil,
				"activeDeployment": nil,
				"bundle": map[string]any{
					"id": "bundle-id",
				},
				"manifest": map[string]any{
					"id": "manifest-id",
				},
				"environment": map[string]any{
					"id": "target-id",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetPackage(t.Context(), &mdClient, pkgName)
	require.NoError(t, err)

	assert.Nil(t, got.DeployedVersion, "DeployedVersion should be nil for never-deployed packages")
	assert.Nil(t, got.LatestDeployment, "LatestDeployment should be nil when not present")
	assert.Nil(t, got.ActiveDeployment, "ActiveDeployment should be nil when not present")
}

func TestGetPackage_WithDeployments(t *testing.T) {
	pkgName := "ecomm-prod-cache"
	version := "0.1.0"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"package": map[string]any{
				"slug":            pkgName,
				"status":          "provisioned",
				"deployedVersion": version,
				"latestDeployment": map[string]any{
					"id":        "deploy-1",
					"status":    "COMPLETED",
					"action":    "PROVISION",
					"version":   version,
					"createdAt": "2026-01-15T10:30:00Z",
				},
				"activeDeployment": map[string]any{
					"id":        "deploy-1",
					"status":    "COMPLETED",
					"action":    "PROVISION",
					"version":   version,
					"createdAt": "2026-01-15T10:30:00Z",
				},
				"bundle": map[string]any{
					"id": "bundle-id",
				},
				"manifest": map[string]any{
					"id": "manifest-id",
				},
				"environment": map[string]any{
					"id": "target-id",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetPackage(t.Context(), &mdClient, pkgName)
	require.NoError(t, err)

	require.NotNil(t, got.DeployedVersion)
	assert.Equal(t, version, *got.DeployedVersion)
	require.NotNil(t, got.LatestDeployment)
	assert.Equal(t, "deploy-1", got.LatestDeployment.ID)
	assert.Equal(t, "COMPLETED", got.LatestDeployment.Status)
	require.NotNil(t, got.ActiveDeployment)
	assert.Equal(t, "deploy-1", got.ActiveDeployment.ID)
}
