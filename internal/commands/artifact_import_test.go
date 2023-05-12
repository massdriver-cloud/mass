package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/spf13/afero"
)

func TestArtifactImport(t *testing.T) {
	client := gqlmock.NewClientWithJSONResponseMap(map[string]interface{}{
		"getArtifactDefinitions": map[string]interface{}{
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

	var fs = afero.NewMemMapFs()

	file, err := fs.Create("artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	file.Write([]byte(`{"name":"fake"}`))

	got, err := commands.ArtifactImport(client, "faux-org-id", fs, "artifact-name", "massdriver/fake-artifact-schema", "artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "artifact-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
