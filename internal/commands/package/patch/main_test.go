package patch_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/package/patch"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestPatchPackage(t *testing.T) {
	params := map[string]interface{}{
		"cidr": "10.0.0.0/16",
	}

	// TODO: this test is bunk, doesnt test anything.
	// Need to add a mock that accepts functions so we can assert what configurePackage receives...
	client := gqlmock.NewClientWithJSONResponseMap(map[string]interface{}{
		"getPackageByNamingConvention": gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
			Manifest: api.Manifest{ID: "manifest-id"},
			Target:   api.Target{ID: "target-id"},
			Params:   params,
		}),
		"configurePackage": map[string]interface{}{
			"data": map[string]interface{}{
				"configurePackage": map[string]interface{}{
					"result": map[string]interface{}{
						"id": "pkg-id",
						"params": map[string]interface{}{
							"cidr": "10.0.0.0/20",
						},
					},
					"successful": true,
				},
			},
		},
	})

	setValues := []string{".cidr = \"10.0.0.0/20\""}

	pkg, err := patch.Run(client, "faux-org-id", "ecomm-prod-cache", setValues)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := map[string]interface{}{
		"cidr": "10.0.0.0/20",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
