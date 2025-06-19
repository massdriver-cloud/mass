package definition_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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
			responses := []any{
				gqlmock.MockMutationResponse("publishArtifactDefinition", api.ArtifactDefinitionWithSchema{
					ID:   "123-456",
					Name: "massdriver/test-schema",
				}),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
			}

			err := definition.Publish(context.Background(), &mdClient, tc.definition)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}
		})
	}
}
