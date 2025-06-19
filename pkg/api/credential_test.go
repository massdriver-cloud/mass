package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestListCredentials(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"artifacts": map[string]any{
				"items": []map[string]any{
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
	mdClient := client.Client{
		GQL: gqlClient,
	}

	credentials, err := api.ListCredentials(t.Context(), &mdClient, "massdriver/aws-iam-role")

	if err != nil {
		t.Fatal(err)
	}

	got := len(credentials)
	want := 2

	if got != want {
		t.Errorf("got %d credentials, wanted %d", got, want)
	}
}
