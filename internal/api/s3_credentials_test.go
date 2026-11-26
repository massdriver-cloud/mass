package api_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestGetS3Credentials(t *testing.T) {
	tests := []struct {
		name     string
		response gqltest.Response
		want     *api.S3Credentials
		wantErr  string
	}{
		{
			name: "returns the credentials",
			response: gqltest.RespondWithData(map[string]any{
				"s3Credentials": map[string]any{
					"accessKeyId":     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
					"secretAccessKey": "secret",
					"endpointUrl":     "https://api.massdriver.cloud/s3",
				},
			}),
			want: &api.S3Credentials{
				AccessKeyID:     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
				SecretAccessKey: "secret",
				EndpointURL:     "https://api.massdriver.cloud/s3",
			},
		},
		{
			name:     "wraps a GraphQL error",
			response: gqltest.RespondWithError("You do not have permission to get S3 credentials without an access token"),
			wantErr:  "get S3 credentials: ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mdClient, mock := newTestClient(t, tc.response)

			got, err := api.GetS3Credentials(t.Context(), mdClient)

			if tc.wantErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want prefix %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetS3Credentials returned an error: %v", err)
			}
			if *got != *tc.want {
				t.Errorf("GetS3Credentials = %+v, want %+v", *got, *tc.want)
			}
			if org := mock.Requests()[0].Variables["organizationId"]; org != "test-org" {
				t.Errorf("organizationId = %v, want test-org", org)
			}
		})
	}
}
