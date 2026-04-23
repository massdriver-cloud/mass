package resourcetype_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestPublish(t *testing.T) {
	type test struct {
		name     string
		path     string
		wantBody string
	}
	tests := []test{
		{
			name:     "simple json",
			path:     "testdata/simple-resource.json",
			wantBody: `{"$schema":"http://json-schema.org/draft-07/schema","type":"object","title":"Test Resource","properties":{"data":{"type":"object"}},"specs":{"type":"object"}}}`,
		},
		{
			name:     "massdriver.yaml format",
			path:     "testdata/massdriver-yaml-simple/massdriver.yaml",
			wantBody: `{"$schema":"http://json-schema.org/draft-07/schema","type":"object","title":"Test Resource"}`,
		},
		{
			name:     "massdriver.yaml with instructions and exports",
			path:     "testdata/massdriver-yaml-resource/massdriver.yaml",
			wantBody: `{"$schema":"http://json-schema.org/draft-07/schema","type":"object","title":"Test Resource"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resourceTypeSchema, err := os.ReadFile("testdata/resourcetype-schema.json")
			if err != nil {
				t.Fatalf("failed to read resource type schema: %v", err)
			}
			metaSchema, err := os.ReadFile("testdata/draft-7.json")
			if err != nil {
				t.Fatalf("failed to read meta schema: %v", err)
			}

			// Start mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/json-schemas/resource-type.json":
					_, _ = w.Write(resourceTypeSchema)
				case "/json-schemas/draft-7.json":
					_, _ = w.Write(metaSchema)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			responses := []any{
				gqlmock.MockMutationResponse("publishResourceType", api.ResourceType{
					ID:   "123-456",
					Name: "massdriver/test-schema",
				}),
			}

			mdClient := client.Client{
				GQLv1: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					URL: server.URL,
				},
			}

			_, err = resourcetype.Publish(t.Context(), &mdClient, tc.path)
			if err != nil {
				t.Fatalf("%v, unexpected error", err)
			}
		})
	}
}
