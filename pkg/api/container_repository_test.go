package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestDockerRegistryToken(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]map[string]string{
			"containerRepository": {
				"token":   "bogustoken",
				"repoUri": "massdriveruswest.pkg.docker.dev",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetContainerRepository(t.Context(), &mdClient, "artifactId", "westus", "massdriver/test-image")
	if err != nil {
		t.Fatal(err)
	}

	want := &api.ContainerRepository{
		Token:         "bogustoken",
		RepositoryURI: "massdriveruswest.pkg.docker.dev",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %v but got %v", want, got)
	}
}
