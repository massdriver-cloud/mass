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
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"createArtifact": map[string]interface{}{
				"result": map[string]interface{}{
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

	got, err := api.CreateArtifact(context.Background(), &mdClient, "artifact-name", "artifact-type", map[string]interface{}{}, map[string]interface{}{})
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
