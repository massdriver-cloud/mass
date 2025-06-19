package definition_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGet(t *testing.T) {
	type test struct {
		name       string
		definition map[string]any
		want       api.ArtifactDefinitionWithSchema
	}
	tests := []test{
		{
			name: "simple",
			definition: map[string]any{
				"id":   "123-456",
				"name": "massdriver/test-schema",
				"schema": map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
				"label": "Test Schema",
			},
			want: api.ArtifactDefinitionWithSchema{
				ID:   "123-456",
				Name: "massdriver/test-schema",
				Schema: map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
				Label: "Test Schema",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockQueryResponse("artifactDefinition", tc.definition),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
			}

			got, err := definition.Get(context.Background(), &mdClient, "massdriver/test-schema")
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %v, want %v", *got, tc.want)
			}
		})
	}
}
