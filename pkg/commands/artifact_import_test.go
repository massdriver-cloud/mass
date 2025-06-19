package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestArtifactImport(t *testing.T) {
	gqlClient := gqlmock.NewClientWithJSONResponseMap(map[string]any{
		"listArtifactDefinitions": map[string]any{
			"data": map[string]any{
				"artifactDefinitions": []map[string]any{
					{
						"name": "massdriver/fake-artifact-schema",
						"schema": map[string]any{
							"$id":     "id",
							"$schema": "http://json-schema.org/draft-07/schema",
							"type":    "object",
							"properties": map[string]any{
								"name": map[string]any{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
		"createArtifact": map[string]any{
			"data": map[string]any{
				"createArtifact": map[string]any{
					"result": map[string]any{
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

	got, err := commands.ArtifactImport(t.Context(), &mdClient, "artifact-name", "massdriver/fake-artifact-schema", "testdata/artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "artifact-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
