package patch_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/package/patch"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
)

func TestPatchPackage(t *testing.T) {
	// Client-side patch gets params, patches, and reconfigures
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) interface{} {
			return gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest: api.Manifest{ID: "manifest-id"},
				Target:   api.Target{ID: "target-id"},
				Params: map[string]interface{}{
					"cidr": "10.0.0.0/16",
				},
			})
		},
		func(req *http.Request) interface{} {
			vars := gqlmock.ParseInputVariables(req)
			paramsJSON := []byte(vars["params"].(string))

			params := map[string]interface{}{}
			gqlmock.MustUnmarshalJSON(paramsJSON, &params)

			return gqlmock.MockMutationResponse("configurePackage", map[string]interface{}{
				"id":     "pkg-id",
				"params": params,
			})
		},
	}

	client := gqlmock.NewClientWithFuncResponseArray(responses)
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
