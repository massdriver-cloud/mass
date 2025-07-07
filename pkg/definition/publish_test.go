package definition_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestPublish(t *testing.T) {
	type test struct {
		name       string
		definition *bytes.Buffer
		wantBody   string
	}
	tests := []test{
		{
			name:       "simple",
			definition: bytes.NewBuffer([]byte(`{"$md":{"access":"public","name":"foo"},"required":["data","specs"],"properties":{"data":{},"specs":{}}}`)),
			wantBody:   `{"$md":{"access":"public","name":"foo"},"required":["data","specs"],"properties":{"data":{},"specs":{}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			artifactDefSchema, err := ioutil.ReadFile("testdata/artdef-schema.json")
			if err != nil {
				t.Fatalf("failed to read artifact definition schema: %v", err)
			}
			metaSchema, err := ioutil.ReadFile("testdata/draft-7.json")
			if err != nil {
				t.Fatalf("failed to read meta schema: %v", err)
			}

			// Start mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/json-schemas/artifact-definition.json":
					w.Write([]byte(artifactDefSchema))
				case "/json-schemas/draft-7.json":
					w.Write([]byte(metaSchema))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			responses := []any{
				gqlmock.MockMutationResponse("publishArtifactDefinition", api.ArtifactDefinitionWithSchema{
					ID:   "123-456",
					Name: "massdriver/test-schema",
				}),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					URL: server.URL,
				},
			}

			_, err = definition.Publish(t.Context(), &mdClient, tc.definition)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}
		})
	}
}
