package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestGetS3Credentials(t *testing.T) {
	mdClient, mock := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"s3Credentials": map[string]any{
			"accessKeyId":     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
			"secretAccessKey": "secret",
			"endpointUrl":     "https://api.massdriver.cloud/s3",
		},
	}))

	got, err := api.GetS3Credentials(t.Context(), mdClient)
	if err != nil {
		t.Fatalf("GetS3Credentials returned an error: %v", err)
	}

	want := api.S3Credentials{
		AccessKeyID:     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
		SecretAccessKey: "secret",
		EndpointURL:     "https://api.massdriver.cloud/s3",
	}
	if *got != want {
		t.Errorf("GetS3Credentials = %+v, want %+v", *got, want)
	}

	if org := mock.Requests()[0].Variables["organizationId"]; org != "test-org" {
		t.Errorf("organizationId = %v, want test-org", org)
	}
}
