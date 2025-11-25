package api_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetArtifactDefinitions(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"artifactDefinition": map[string]any{
				"name": "massdriver/aws-ecs-cluster",
				"schema": map[string]any{
					"properties": map[string]any{
						"aws_authentication": map[string]string{
							"type": "object",
						},
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetArtifactDefinition(t.Context(), &mdClient, "massdriver/aws-ecs-cluster")
	if err != nil {
		t.Fatal(err)
	}

	want := api.ArtifactDefinitionWithSchema{
		Name: "massdriver/aws-ecs-cluster",
		Schema: map[string]any{
			"properties": map[string]any{
				"aws_authentication": map[string]any{
					"type": "object",
				},
			},
		},
	}

	gqlClient.AssertQueryCalled(t, "artifactDefinition", map[string]any{
		"name": "massdriver/aws-ecs-cluster",
	})

	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %v expected %v", *got, want)
	}
}

func TestListArtifactDefinitions(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"artifactDefinitions": []map[string]any{
				{
					"name": "massdriver/aws-ecs-cluster",
					"schema": map[string]any{
						"properties": map[string]any{
							"aws_authentication": map[string]string{
								"type": "object",
							},
						},
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.ListArtifactDefinitions(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	want := []api.ArtifactDefinitionWithSchema{
		{
			Name: "massdriver/aws-ecs-cluster",
			Schema: map[string]any{
				"properties": map[string]any{
					"aws_authentication": map[string]any{
						"type": "object",
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(got[0].Schema, want[0].Schema) {
		t.Errorf("got %v expected %v", got[0].Schema, want[0].Schema)
	}
}

func TestPublishArtifactDefinition(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return gqlmock.MockMutationResponse("publishArtifactDefinition", map[string]any{
				"name": "massdriver/test-schema",
				"id":   "123-456",
			})
		},
	}

	artDef := map[string]any{
		"$id":  "123-456",
		"name": "massdriver/test-schema",
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithFuncResponseArray(responses),
	}

	got, err := api.PublishArtifactDefinition(t.Context(), &mdClient, artDef)
	if err != nil {
		t.Fatal(err)
	}

	want := api.ArtifactDefinitionWithSchema{
		ID:   "123-456",
		Name: "massdriver/test-schema",
	}

	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %v, wanted %v", *got, want)
	}
}
