package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestDockerRegistryToken(t *testing.T) {
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]map[string]string{
			"containerRepository": {
				"token":   "bogustoken",
				"repoUri": "massdriveruswest.pkg.docker.dev",
			},
		},
	})

	got, err := api.GetContainerRepository(client, "artifactId", "orgId", "westus", "massdriver/test-image")

	if err != nil {
		t.Fatal(err)
	}

	want := &api.ContainerRepository{
		Token:         "bogustoken",
		RepositoryUri: "massdriveruswest.pkg.docker.dev",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %v but got %v", want, got)
	}
}
