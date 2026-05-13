package resourcetype_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestPublish(t *testing.T) {
	type test struct {
		name string
		path string
	}
	tests := []test{
		{
			name: "simple json",
			path: "testdata/simple-resource.json",
		},
		{
			name: "massdriver.yaml format",
			path: "testdata/massdriver-yaml-simple/massdriver.yaml",
		},
		{
			name: "massdriver.yaml with instructions and exports",
			path: "testdata/massdriver-yaml-resource/massdriver.yaml",
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

			// Start mock HTTP server (serves the meta-schema and the resource-type
			// JSON Schema that Publish() validates the input against).
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

			mock := gqltest.NewClient(
				gqltest.RespondWithData(map[string]any{
					"publishResourceType": map[string]any{
						"result": map[string]any{
							"id":   "123-456",
							"name": "massdriver/test-schema",
						},
						"successful": true,
					},
				}),
			)
			t.Cleanup(api.SetTransportForTest(mock))

			mdClient, err := massdriver.NewClient(
				massdriver.WithGQLClient(mock),
				massdriver.WithBaseURL(server.URL),
				massdriver.WithOrganizationID("test-org"),
			)
			if err != nil {
				t.Fatal(err)
			}

			_, err = resourcetype.Publish(t.Context(), mdClient, tc.path)
			if err != nil {
				t.Fatalf("%v, unexpected error", err)
			}
		})
	}
}
