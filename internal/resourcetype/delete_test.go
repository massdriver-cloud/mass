package resourcetype_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestDelete(t *testing.T) {
	type test struct {
		name       string
		typeName   string
		response   map[string]any
		force      bool
		expectErr  bool
		errMessage string
	}
	tests := []test{
		{
			name:     "simple",
			typeName: "aws-s3",
			response: map[string]any{
				"id":   "123-456",
				"name": "massdriver/test-schema",
			},
			force: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := gqltest.NewClient(
				// Delete first does a Get for the confirmation prompt.
				gqltest.RespondWithData(map[string]any{
					"resourceType": tc.response,
				}),
				gqltest.RespondWithData(map[string]any{
					"deleteResourceType": map[string]any{
						"result":     tc.response,
						"successful": true,
					},
				}),
			)
			t.Cleanup(api.SetTransportForTest(mock))
			mdClient, err := massdriver.NewClient(
				massdriver.WithGQLClient(mock),
				massdriver.WithOrganizationID("org-123"),
			)
			if err != nil {
				t.Fatal(err)
			}

			err = resourcetype.Delete(t.Context(), mdClient, tc.typeName, tc.force)
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
