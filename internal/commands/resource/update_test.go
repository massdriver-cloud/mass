package resource_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/resource"
	"github.com/massdriver-cloud/mass/internal/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestResourceUpdate(t *testing.T) {
	gqlClient := gqlmock.NewClientWithJSONResponseMap(map[string]any{
		"updateResource": map[string]any{
			"data": map[string]any{
				"updateResource": map[string]any{
					"result": map[string]any{
						"id":   "resource-id",
						"name": "resource-name",
					},
					"successful": true,
				},
			},
		},
	})

	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	got, err := resource.RunUpdate(t.Context(), &mdClient, "resource-id", "resource-name", "testdata/resource.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "resource-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}

func TestResourceUpdateWithoutName(t *testing.T) {
	// When no name is provided, RunUpdate fetches the existing resource first.
	responses := []any{
		gqlmock.MockQueryResponse("resource", map[string]any{
			"id":      "resource-id",
			"name":    "existing-name",
			"type":    "massdriver/aws-s3",
			"field":   "",
			"payload": map[string]any{},
			"formats": []string{},
			"origin":  "IMPORTED",
			"resourceDefinition": map[string]any{
				"id":    "def-id",
				"name":  "aws-s3",
				"label": "AWS S3",
			},
		}),
		map[string]any{
			"data": map[string]any{
				"updateResource": map[string]any{
					"result": map[string]any{
						"id":   "resource-id",
						"name": "existing-name",
					},
					"successful": true,
				},
			},
		},
	}

	mdClient := client.Client{
		GQLv1: gqlmock.NewClientWithJSONResponseArray(responses),
	}

	got, err := resource.RunUpdate(t.Context(), &mdClient, "resource-id", "", "testdata/resource.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "resource-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
