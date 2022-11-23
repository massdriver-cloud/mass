package api_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
)

func TestGetPackage(t *testing.T) {
	pkgName := "ecomm-prod-cache"

	client := mockClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"getPackageByNamingConvention": map[string]interface{}{
				"namePrefix": fmt.Sprintf("%s-0000", pkgName),
			},
		},
	})

	pkg, err := api.GetPackage(client, "faux-org-id", pkgName)

	if err != nil {
		t.Fatal(err)
	}

	if got, want := pkg.NamePrefix, "ecomm-prod-cache-0000"; got != want {
		t.Errorf("got pkg.NamePrefix: %q, want: %q", got, want)
	}
}

func TestDeployPackage(t *testing.T) {
	want := "pkg-uuid1"
	client := mockClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployPackage": map[string]interface{}{
				"result": map[string]interface{}{
					"id": want,
				},
				"successful": true,
			},
		},
	})

	pkg, err := api.DeployPackage(client, "faux-org-id", "target-id", "manifest-id")

	if err != nil {
		t.Fatal(err)
	}

	got := pkg.ID

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
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
