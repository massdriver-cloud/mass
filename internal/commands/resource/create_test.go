package resource_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/resource"
	"github.com/massdriver-cloud/mass/internal/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestResourceImport(t *testing.T) {
	gqlClient := gqlmock.NewClientWithJSONResponseMap(map[string]any{
		"getResourceType": map[string]any{
			"data": map[string]any{
				"resourceType": map[string]any{
					"id":   "massdriver/fake-resource-schema",
					"name": "massdriver/fake-resource-schema",
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
		"createResource": map[string]any{
			"data": map[string]any{
				"createResource": map[string]any{
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
		GQLv2: gqlClient,
	}

	got, err := resource.RunCreate(t.Context(), &mdClient, "resource-name", "massdriver/fake-resource-schema", "testdata/resource.json")

	if err != nil {
		t.Fatal(err)
	}

	want := "resource-id"
	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
