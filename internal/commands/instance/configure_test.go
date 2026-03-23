package instance_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunConfigure(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackage", api.Package{
				Manifest:    &api.Manifest{ID: "manifest-id"},
				Environment: &api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			vars := gqlmock.ParseInputVariables(req)
			paramsStr, ok := vars["params"].(string)
			if !ok {
				panic("vars[\"params\"] is not a string")
			}
			paramsJSON := []byte(paramsStr)

			params := map[string]any{}
			gqlmock.MustUnmarshalJSON(paramsJSON, &params)

			return gqlmock.MockMutationResponse("configurePackage", map[string]any{
				"id":     "instance-id",
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

	instance, err := instance.RunConfigure(t.Context(), &mdClient, "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := instance.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestConfigurePackageInterpolation(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockQueryResponse("getPackage", api.Package{
				Manifest:    &api.Manifest{ID: "manifest-id"},
				Environment: &api.Environment{ID: "target-id"},
			})
		},
		func(req *http.Request) any {
			vars := gqlmock.ParseInputVariables(req)
			paramsStr, ok := vars["params"].(string)
			if !ok {
				panic("vars[\"params\"] is not a string")
			}
			paramsJSON := []byte(paramsStr)

			params := map[string]any{}
			gqlmock.MustUnmarshalJSON(paramsJSON, &params)

			return gqlmock.MockMutationResponse("configurePackage", map[string]any{
				"id":     "instance-id",
				"params": params,
			})
		},
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}
	params := map[string]any{"size": "${MEMORY_AMT}GB"}
	t.Setenv("MEMORY_AMT", "6")

	instance, err := instance.RunConfigure(t.Context(), &mdClient, "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := instance.Params
	want := map[string]any{
		"size": "6GB",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
