package configure_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/package/configure"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestConfigurePackage(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest:    api.Manifest{ID: "manifest-id"},
				Environment: api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			vars := gqlmock.ParseInputVariables(req)
			paramsJSON := []byte(vars["params"].(string))

			params := map[string]any{}
			gqlmock.MustUnmarshalJSON(paramsJSON, &params)

			return gqlmock.MockMutationResponse("configurePackage", map[string]any{
				"id":     "pkg-id",
				"params": params,
			})
		},
	}

	params := map[string]any{
		"cidr": "10.0.0.0/16",
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}

	pkg, err := configure.Run(context.Background(), &mdClient, "ecomm-prod-cache", params)
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
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest:    api.Manifest{ID: "manifest-id"},
				Environment: api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			vars := gqlmock.ParseInputVariables(req)
			paramsJSON := []byte(vars["params"].(string))

			params := map[string]any{}
			gqlmock.MustUnmarshalJSON(paramsJSON, &params)

			return gqlmock.MockMutationResponse("configurePackage", map[string]any{
				"id":     "pkg-id",
				"params": params,
			})
		},
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}
	params := map[string]any{"size": "${MEMORY_AMT}GB"}
	t.Setenv("MEMORY_AMT", "6")

	pkg, err := configure.Run(context.Background(), &mdClient, "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := map[string]any{
		"size": "6GB",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
