package api_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
)

func TestGetPackageByName(t *testing.T) {
	pkgName := "ecomm-prod-cache"

	client := mockClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"getPackageByNamingConvention": map[string]interface{}{
				"namePrefix": fmt.Sprintf("%s-0000", pkgName),
				"manifest": map[string]interface{}{
					"id": "manifest-id",
				},
				"target": map[string]interface{}{
					"id": "target-id",
				},
			},
		},
	})

	got, err := api.GetPackageByName(client, "faux-org-id", pkgName)

	if err != nil {
		t.Fatal(err)
	}

	want := &api.Package{
		NamePrefix: "ecomm-prod-cache-0000",
		Manifest: api.Manifest{
			ID: "manifest-id",
		},
		Target: api.Target{
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

	client := mockClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"configurePackage": map[string]interface{}{
				"result": map[string]interface{}{
					"id":     "pkg-uuid1",
					"params": string(mustMarshalJSON(params)),
				},
				"successful": true,
			},
		},
	})

	pkg, err := api.ConfigurePackage(client, "faux-org-id", "faux-target-id", "faux-manifest-id", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
