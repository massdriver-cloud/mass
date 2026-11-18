package credentials_test

import (
	"bytes"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/credentials"
)

const exampleToken = "md_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

func TestNewS3DerivesKeysFromTheAccessToken(t *testing.T) {
	creds, err := credentials.NewS3(exampleToken, "https://api.massdriver.cloud/")
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	if creds.AccessKeyID != "md_abcdefghi" {
		t.Errorf("AccessKeyID = %q, want %q", creds.AccessKeyID, "md_abcdefghi")
	}
	if creds.SecretAccessKey != "8ccb69353e7475180dcb95cf74b7295f1d4335498395ae620317bc09f09ccc58" {
		t.Errorf("SecretAccessKey = %q", creds.SecretAccessKey)
	}
	if creds.EndpointURL != "https://api.massdriver.cloud/s3" {
		t.Errorf("EndpointURL = %q, want %q", creds.EndpointURL, "https://api.massdriver.cloud/s3")
	}
}

func TestNewS3RejectsCredentialsThatAreNotAccessTokens(t *testing.T) {
	if _, err := credentials.NewS3("mds_session_token_value", "https://api.massdriver.cloud"); err == nil {
		t.Error("NewS3 accepted a session token, want an error")
	}
}

func TestRunS3WritesCredentialProcessJSON(t *testing.T) {
	creds := credentials.S3{AccessKeyID: "md_abcdefghi", SecretAccessKey: "secret", EndpointURL: "https://api.massdriver.cloud/s3"}

	var out bytes.Buffer
	if err := credentials.RunS3(creds, "process", &out); err != nil {
		t.Fatalf("RunS3: %v", err)
	}

	want := `{"Version":1,"AccessKeyId":"md_abcdefghi","SecretAccessKey":"secret"}` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestRunS3WritesShellExports(t *testing.T) {
	creds := credentials.S3{AccessKeyID: "md_abcdefghi", SecretAccessKey: "secret", EndpointURL: "https://api.massdriver.cloud/s3"}

	var out bytes.Buffer
	if err := credentials.RunS3(creds, "env", &out); err != nil {
		t.Fatalf("RunS3: %v", err)
	}

	want := "export AWS_ACCESS_KEY_ID=md_abcdefghi\n" +
		"export AWS_SECRET_ACCESS_KEY=secret\n" +
		"export AWS_ENDPOINT_URL_S3=https://api.massdriver.cloud/s3\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
