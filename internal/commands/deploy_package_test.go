package commands_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestDeployPackage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(mockEndpoint, func(w http.ResponseWriter, req *http.Request) {
		var parsedReq graphQLRequest
		json.NewDecoder(req.Body).Decode(&parsedReq)

		// TODO: map of responses helper & array helper.
		responses := map[string]interface{}{
			"getPackageByNamingConvention": mockQueryResponse("getPackageByNamingConvention", api.Package{
				Manifest: api.Manifest{ID: "manifest-id"},
				Target:   api.Target{ID: "target-id"},
			}),
			"deployPackage": mockMutationResponse("deployPackage", api.Deployment{
				ID:     "deployment-id",
				Status: "STARTED",
			}),
			"getDeploymentById": mockQueryResponse("deployment", api.Deployment{
				ID:     "deployment-id",
				Status: "PENDING",
			}),
		}

		response := responses[parsedReq.OperationName]

		fmt.Printf("\nRESPONSE: %s \n\t%v", parsedReq.OperationName, response)
		data, _ := json.Marshal(response)

		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, string(data))
	})

	client := mockClient(mux)

	deployment, err := commands.DeployPackage(client, "faux-org-id", "ecomm-prod-cache")
	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "COMPLETED"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
