package definition_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestDelete(t *testing.T) {
	type test struct {
		name       string
		defName    string
		response   map[string]any
		wantID     string
		wantName   string
		force      bool
		expectErr  bool
		errMessage string
	}
	tests := []test{
		{
			name:    "simple",
			defName: "aws-s3",
			response: map[string]any{
				"id":   "123-456",
				"name": "massdriver/test-schema",
			},
			wantID:   "def-123",
			wantName: "org-123/aws-s3",
			force:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockQueryResponse("getArtifactDefinition", tc.response),
				gqlmock.MockMutationResponse("deleteArtifactDefinition", tc.response),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					OrganizationID: "org-123",
				},
			}

			err := definition.Delete(t.Context(), &mdClient, tc.defName, tc.force)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if !strings.Contains(err.Error(), tc.errMessage) {
					t.Fatalf("expected error message to contain %q but got %q", tc.errMessage, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
