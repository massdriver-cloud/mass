package preview_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/preview"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestDeployPreviewEnvironment(t *testing.T) {
	projectSlug := "ecomm"
	envSlug := "p9000"
	responses := []any{
		gqlmock.MockMutationResponse("deployPreviewEnvironment", map[string]any{
			"id":   "envUUID",
			"slug": envSlug,
			"project": map[string]any{
				"id":   "projUUID",
				"slug": projectSlug,
			},
		}),
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithJSONResponseArray(responses),
		Config: config.Config{
			OrganizationID: "faux-org-id",
		},
	}

	previewCfg := api.PreviewConfig{
		ProjectSlug: "fake-project-slug",
		Credentials: []api.Credential{},
		Packages:    make(map[string]api.PreviewPackage),
	}

	ciContext := map[string]any{}

	env, err := preview.RunDeploy(t.Context(), &mdClient, projectSlug, &previewCfg, &ciContext)

	if err != nil {
		t.Fatal(err)
	}

	got := env.URL
	want := "https://app.massdriver.cloud/orgs/faux-org-id/projects/projUUID/targets/envUUID"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestDeployPreviewEnvironmentInterpolation(t *testing.T) {
	projectSlug := "ecomm"
	envSlug := "p9000"

	mux := http.NewServeMux()
	mux.HandleFunc(gqlmock.MockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq gqlmock.GraphQLRequest
		if err := json.NewDecoder(req.Body).Decode(&parsedReq); err != nil {
			t.Error(err)
		}

		input := parsedReq.Variables["input"]
		inputMap, ok := input.(map[string]any)
		_ = ok

		paramsJSON := []byte((inputMap["packageConfigurations"]).(string))

		got := map[string]any{}
		gqlmock.MustUnmarshalJSON(paramsJSON, &got)

		want := map[string]any{
			"myApp": map[string]any{
				"params": map[string]any{
					"hostname": "preview-9000.example.com",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v", got, want)
		}

		response := gqlmock.MockMutationResponse("deployPreviewEnvironment", map[string]any{
			"id":   "envUUID",
			"slug": envSlug,
			"project": map[string]any{
				"id":   "projUUID",
				"slug": projectSlug,
			},
		})

		data, _ := json.Marshal(response)
		gqlmock.MustWrite(w, string(data))
	})

	mdClient := client.Client{
		GQL: gqlmock.NewClient(mux),
	}

	previewCfg := api.PreviewConfig{
		ProjectSlug: "",
		Credentials: []api.Credential{},
		Packages: map[string]api.PreviewPackage{
			"myApp": {
				Params: map[string]any{
					"hostname": "preview-${PR_NUMBER}.example.com",
				},
			},
		},
	}

	ciContext := map[string]any{}

	t.Setenv("PR_NUMBER", "9000")
	_, err := preview.RunDeploy(t.Context(), &mdClient, projectSlug, &previewCfg, &ciContext)

	if err != nil {
		t.Fatal(err)
	}
}
