package api_test

import (
	"fmt"
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
				"namePrefix": fmt.Sprintf("%s-0000", pkgName),
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

	want := &api.Package{
		NamePrefix: "ecomm-prod-cache-0000",
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
