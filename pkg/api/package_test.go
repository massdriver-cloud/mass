package api_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetPackageByName(t *testing.T) {
	pkgName := "ecomm-prod-cache"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"getPackageByNamingConvention": map[string]interface{}{
				"namePrefix": fmt.Sprintf("%s-0000", pkgName),
				"manifest": map[string]interface{}{
					"id": "manifest-id",
				},
				"environment": map[string]interface{}{
					"id": "target-id",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetPackageByName(context.Background(), &mdClient, pkgName)

	if err != nil {
		t.Fatal(err)
	}

	want := &api.Package{
		NamePrefix: "ecomm-prod-cache-0000",
		Manifest: api.Manifest{
			ID: "manifest-id",
		},
		Environment: api.Environment{
			ID: "target-id",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestConfigurePackage(t *testing.T) {
	params := map[string]interface{}{
		"cidr": "10.0.0.0/16",
	}

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"configurePackage": map[string]interface{}{
				"result": map[string]interface{}{
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

	pkg, err := api.ConfigurePackage(context.Background(), &mdClient, "faux-target-id", "faux-manifest-id", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
