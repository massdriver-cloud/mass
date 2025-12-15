package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestDeleteArtifactDefinition(t *testing.T) {
	type test struct {
		name     string
		defName  string
		response map[string]any
		want     api.ArtifactDefinitionWithSchema
	}
	tests := []test{
		{
			name:    "simple",
			defName: "aws-s3",
			response: map[string]any{
				"id":   "def-123",
				"name": "org-123/aws-s3",
			},
			want: api.ArtifactDefinitionWithSchema{
				ID:   "def-123",
				Name: "org-123/aws-s3",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockMutationResponse("deleteArtifactDefinition", tc.response),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					OrganizationID: "org-123",
				},
			}

			got, err := api.DeleteArtifactDefinition(t.Context(), &mdClient, tc.defName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != tc.want.ID {
				t.Errorf("got ID %v, want %v", got.ID, tc.want.ID)
			}
			if got.Name != tc.want.Name {
				t.Errorf("got Name %v, want %v", got.Name, tc.want.Name)
			}
		})
	}
}
