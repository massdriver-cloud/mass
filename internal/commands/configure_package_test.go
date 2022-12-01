package commands_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestConfigurePackage(t *testing.T) {
	params := map[string]interface{}{
		"cidr": "10.0.0.0/16",
	}

	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq graphQLRequest
		json.NewDecoder(req.Body).Decode(&parsedReq)

		responses := map[string]interface{}{
			"getPackageByNamingConvention": mockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest: api.Manifest{ID: "manifest-id"},
				Target:   api.Target{ID: "target-id"},
			}),
			"configurePackage": map[string]interface{}{
				"data": map[string]interface{}{
					"configurePackage": map[string]interface{}{
						"result": map[string]interface{}{
							"id":     "pkg-id",
							"params": string(mustMarshalJSON(params)),
						},
						"successful": true,
					},
				},
			},
		}

		response := responses[parsedReq.OperationName]
		data, _ := json.Marshal(response)

		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, string(data))
	})

	client := mockClient(mux)

	pkg, err := commands.ConfigurePackage(client, "faux-org-id", "ecomm-prod-cache", params)
	if err != nil {
		t.Fatal(err)
	}

	got := pkg.Params
	want := params

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
