package credentials_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/credentials"
)

type fakeAPI struct {
	creds *api.S3Credentials
	err   error
	calls int
}

func (f *fakeAPI) GetS3Credentials(context.Context) (*api.S3Credentials, error) {
	f.calls++
	return f.creds, f.err
}

func TestRunS3(t *testing.T) {
	example := &api.S3Credentials{
		AccessKeyID:     "0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10",
		SecretAccessKey: "secret",
		EndpointURL:     "https://api.massdriver.cloud/s3",
	}

	tests := []struct {
		name      string
		creds     *api.S3Credentials
		apiErr    error
		output    string
		want      string
		wantErr   string
		wantCalls int
	}{
		{
			name:      "credential_process JSON",
			creds:     example,
			output:    "process",
			want:      `{"Version":1,"AccessKeyId":"0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10","SecretAccessKey":"secret"}` + "\n",
			wantCalls: 1,
		},
		{
			name:   "shell exports",
			creds:  example,
			output: "env",
			want: "export AWS_ACCESS_KEY_ID='0b5f0a52-4a3c-4f63-9d0e-2f3b1c7e8a10'\n" +
				"export AWS_SECRET_ACCESS_KEY='secret'\n" +
				"export AWS_ENDPOINT_URL_S3='https://api.massdriver.cloud/s3'\n",
			wantCalls: 1,
		},
		{
			name:   "shell exports keep shell syntax in values literal",
			creds:  &api.S3Credentials{AccessKeyID: "a'b", SecretAccessKey: "$(rm -rf ~)", EndpointURL: "x; y"},
			output: "env",
			want: "export AWS_ACCESS_KEY_ID='a'\\''b'\n" +
				"export AWS_SECRET_ACCESS_KEY='$(rm -rf ~)'\n" +
				"export AWS_ENDPOINT_URL_S3='x; y'\n",
			wantCalls: 1,
		},
		{
			name:      "unsupported output fails before the API call",
			creds:     example,
			output:    "yaml",
			wantErr:   `unsupported output format "yaml": use one of process, env`,
			wantCalls: 0,
		},
		{
			name:      "API error",
			apiErr:    errors.New("get S3 credentials: forbidden"),
			output:    "process",
			wantErr:   "get S3 credentials: forbidden",
			wantCalls: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeAPI{creds: tc.creds, err: tc.apiErr}

			var out bytes.Buffer
			err := credentials.RunS3(t.Context(), fake, tc.output, &out)

			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("RunS3: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("output = %q, want %q", out.String(), tc.want)
			}
			if fake.calls != tc.wantCalls {
				t.Errorf("API calls = %d, want %d", fake.calls, tc.wantCalls)
			}
		})
	}
}
