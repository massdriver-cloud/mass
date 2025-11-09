package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetPackageByName(t *testing.T) {
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

	got, err := api.GetPackageByName(t.Context(), &mdClient, pkgName)

	if err != nil {
		t.Fatal(err)
	}

	// Compare values, not pointers
	if got.Slug != "ecomm-prod-cache" {
		t.Errorf("got Slug %s, wanted ecomm-prod-cache", got.Slug)
	}
	if got.Bundle == nil || got.Bundle.ID != "bundle-id" {
		t.Errorf("got Bundle %v, wanted bundle-id", got.Bundle)
	}
	if got.Manifest == nil || got.Manifest.ID != "manifest-id" {
		t.Errorf("got Manifest %v, wanted manifest-id", got.Manifest)
	}
	if got.Environment == nil || got.Environment.ID != "target-id" {
		t.Errorf("got Environment %v, wanted target-id", got.Environment)
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
