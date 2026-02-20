package artifact_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/artifact"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestArtifactUpdate(t *testing.T) {
	gqlClient := gqlmock.NewClientWithJSONResponseMap(map[string]any{
		"updateArtifact": map[string]any{
			"data": map[string]any{
				"updateArtifact": map[string]any{
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

	got, err := artifact.RunUpdate(t.Context(), &mdClient, "artifact-id", "artifact-name", "testdata/artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "artifact-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestArtifactUpdateWithoutName(t *testing.T) {
	// When no name is provided, RunUpdate fetches the existing artifact first.
	responses := []any{
		gqlmock.MockQueryResponse("artifact", map[string]any{
			"id":      "artifact-id",
			"name":    "existing-name",
			"type":    "massdriver/aws-s3",
			"field":   "",
			"payload": map[string]any{},
			"formats": []string{},
			"origin":  "IMPORTED",
			"artifactDefinition": map[string]any{
				"id":    "def-id",
				"name":  "aws-s3",
				"label": "AWS S3",
			},
		}),
		map[string]any{
			"data": map[string]any{
				"updateArtifact": map[string]any{
					"result": map[string]any{
						"id":   "artifact-id",
						"name": "existing-name",
					},
					"successful": true,
				},
			},
		},
	}

	mdClient := client.Client{
		GQL: gqlmock.NewClientWithJSONResponseArray(responses),
	}

	got, err := artifact.RunUpdate(t.Context(), &mdClient, "artifact-id", "", "testdata/artifact.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "artifact-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
