package commands_test

import (
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestArtifactImport(t *testing.T) {
	gqlClient := gqlmock.NewClientWithJSONResponseMap(map[string]interface{}{
		"listArtifactDefinitions": map[string]interface{}{
			"data": map[string]interface{}{
				"artifactDefinitions": []map[string]interface{}{
					{
						"name": "massdriver/fake-artifact-schema",
						"schema": map[string]interface{}{
							"$id":     "id",
							"$schema": "http://json-schema.org/draft-07/schema",
							"type":    "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
		"createArtifact": map[string]interface{}{
			"data": map[string]interface{}{
				"createArtifact": map[string]interface{}{
					"result": map[string]interface{}{
						"id":   "artifact-id",
						"name": "artifact-name",
					},
					"successful": true,
				},
			},
		},
	})

	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := commands.ArtifactImport(context.Background(), &mdClient, "artifact-name", "massdriver/fake-artifact-schema", "testdata/artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "artifact-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
