package commands_test

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestDeployPreviewEnvironment(t *testing.T) {
	projectSlug := "ecomm"
	envSlug := "p9000"
	responses := []interface{}{
		mockMutationResponse("deployPreviewEnvironment", map[string]interface{}{
			"id":   "envUUID",
			"slug": envSlug,
			"project": map[string]interface{}{
				"id":   "projUUID",
				"slug": projectSlug,
			},
		}),
	}

	client := mockClientWithJSONResponseArray(responses)

	previewCfg := commands.PreviewConfig{
		Credentials:   map[string]string{},
		PackageParams: map[string]interface{}{},
	}

	ciContext := map[string]interface{}{}

	env, err := commands.DeployPreviewEnvironment(client, "faux-org-id", projectSlug, &previewCfg, &ciContext)

	if err != nil {
		t.Fatal(err)
	}

	got := env.URL
	want := "https://app.massdriver.cloud/projects/projUUID/targets/envUUID"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestDeployPreviewEnvironmentInterpolation(t *testing.T) {
	projectSlug := "ecomm"
	envSlug := "p9000"

	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq graphQLRequest
		if err := json.NewDecoder(req.Body).Decode(&parsedReq); err != nil {
			t.Error(err)
		}

		input := (parsedReq.Variables["input"]).(map[string]interface{})
		paramsJSON := []byte((input["packageParams"]).(string))

		got := map[string]interface{}{}
		mustUnmarshalJSON(paramsJSON, &got)

		want := map[string]interface{}{
			"myApp": map[string]interface{}{
				"hostname": "preview-9000.example.com",
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, wanted %v", got, want)
		}

		response := mockMutationResponse("deployPreviewEnvironment", map[string]interface{}{
			"id":   "envUUID",
			"slug": envSlug,
			"project": map[string]interface{}{
				"id":   "projUUID",
				"slug": projectSlug,
			},
		})

		data, _ := json.Marshal(response)
		mustWrite(w, string(data))
	})

	client := mockClient(mux)

	previewCfg := commands.PreviewConfig{
		Credentials: map[string]string{},
		PackageParams: map[string]interface{}{
			"myApp": map[string]interface{}{
				"hostname": "preview-${PR_NUMBER}.example.com",
			},
		},
	}

	ciContext := map[string]interface{}{}

	os.Setenv("PR_NUMBER", "9000")
	_, err := commands.DeployPreviewEnvironment(client, "faux-org-id", projectSlug, &previewCfg, &ciContext)

	if err != nil {
		t.Fatal(err)
	}
}
