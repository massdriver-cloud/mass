package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
)

func TestGetArtifactDefinitions(t *testing.T) {
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"artifactDefinitions": []map[string]interface{}{
				{
					"name": "massdriver/aws-ecs-cluster",
					"schema": map[string]interface{}{
						"properties": map[string]interface{}{
							"aws_authentication": map[string]string{
								"type": "object",
							},
						},
					},
				},
			},
		},
	})

	got, err := api.GetArtifactDefinitions(client, "faux-org-id")

	if err != nil {
		t.Fatal(err)
	}

	want := []api.ArtifactDefinitionWithSchema{
		{
			Name: "massdriver/aws-ecs-cluster",
			Schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"aws_authentication": map[string]interface{}{
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
