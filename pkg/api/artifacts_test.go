package api_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestCreateArtifact(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createArtifact": map[string]any{
				"result": map[string]any{
					"id":   "artifact-id",
					"name": "artifact-name",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.CreateArtifact(context.Background(), &mdClient, "artifact-name", "artifact-type", map[string]any{}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	want := &api.Artifact{
		Name: "artifact-name",
		ID:   "artifact-id",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %v but got %v", want, got)
	}
}
