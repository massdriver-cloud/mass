package credentials_test

import (
	"bytes"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/credentials"
)

var exampleCredentials = api.S3Credentials{
	AccessKeyID:     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
	SecretAccessKey: "secret",
	EndpointURL:     "https://api.massdriver.cloud/s3",
}

func TestRunS3WritesCredentialProcessJSON(t *testing.T) {
	var out bytes.Buffer
	if err := credentials.RunS3(exampleCredentials, "process", &out); err != nil {
		t.Fatalf("RunS3: %v", err)
	}

	want := `{"Version":1,"AccessKeyId":"0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10","SecretAccessKey":"secret"}` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestRunS3WritesShellExports(t *testing.T) {
	var out bytes.Buffer
	if err := credentials.RunS3(exampleCredentials, "env", &out); err != nil {
		t.Fatalf("RunS3: %v", err)
	}

	want := "export AWS_ACCESS_KEY_ID=0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10\n" +
		"export AWS_SECRET_ACCESS_KEY=secret\n" +
		"export AWS_ENDPOINT_URL_S3=https://api.massdriver.cloud/s3\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
