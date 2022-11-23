package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
)

func TestListCredentials(t *testing.T) {
	mux := muxWithJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"artifacts": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":   "uuid1",
						"name": "artifact1",
					},
					{
						"id":   "uuid2",
						"name": "artifact2",
					},
				},
			},
		},
	})

	client := mockClient(mux)
	credentials, err := api.ListCredentials(client, "faux-org-id", "massdriver/aws-iam-role")

	if err != nil {
		t.Fatal(err)
	}

	got := len(credentials)
	want := 2

	if got != want {
		t.Errorf("got %d credentials, wanted %d", got, want)
	}
}
