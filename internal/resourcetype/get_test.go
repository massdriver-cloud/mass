package resourcetype_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGet(t *testing.T) {
	type test struct {
		name         string
		resourceType map[string]any
		want         api.ResourceType
	}
	tests := []test{
		{
			name: "simple",
			resourceType: map[string]any{
				"id":   "123-456",
				"name": "massdriver/test-schema",
				"schema": map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
			},
			want: api.ResourceType{
				ID:   "123-456",
				Name: "massdriver/test-schema",
				Schema: map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockQueryResponse("resourceType", tc.resourceType),
			}

			mdClient := client.Client{
				GQLv1: gqlmock.NewClientWithJSONResponseArray(responses),
			}

			got, err := resourcetype.Get(t.Context(), &mdClient, "massdriver/test-schema")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %v, want %v", *got, tc.want)
			}
		})
	}
}
