package configure_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/package/configure"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestConfigurePackage(t *testing.T) {
	params := map[string]interface{}{
		"cidr": "10.0.0.0/16",
	}

	client := gqlmock.NewClientWithJSONResponseMap(map[string]interface{}{
		"getPackageByNamingConvention": gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
			Manifest: api.Manifest{ID: "manifest-id"},
			Target:   api.Target{ID: "target-id"},
		}),
		"configurePackage": map[string]interface{}{
			"data": map[string]interface{}{
				"configurePackage": map[string]interface{}{
					"result": map[string]interface{}{
						"id":     "pkg-id",
						"params": params,
					},
					"successful": true,
				},
			},
		},
	})

	pkg, err := configure.Run(client, "faux-org-id", "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
