package configure_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/package/configure"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestConfigurePackage(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) interface{} {
			return gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest: api.Manifest{ID: "manifest-id"},
				Target:   api.Target{ID: "target-id"},
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

	params := map[string]interface{}{
		"cidr": "10.0.0.0/16",
	}

	client := gqlmock.NewClientWithFuncResponseArray(responses)

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

func TestConfigurePackageInterpolation(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) interface{} {
			return gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest: api.Manifest{ID: "manifest-id"},
				Target:   api.Target{ID: "target-id"},
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

	params := map[string]interface{}{"size": "${MEMORY_AMT}GB"}
	t.Setenv("MEMORY_AMT", "6")

	pkg, err := configure.Run(client, "faux-org-id", "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := map[string]interface{}{
		"size": "6GB",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
