// Package credentials prints credentials that other tools use to reach Massdriver.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// API is the narrow surface the credentials commands use. Tests supply a
// hand-rolled fake; production callers use [NewAPI] to bind a *massdriver.Client.
type API interface {
	GetS3Credentials(ctx context.Context) (*api.S3Credentials, error)
}

// NewAPI returns the production [API] backed by the SDK client.
func NewAPI(c *massdriver.Client) API { return sdkAPI{c: c} }

type sdkAPI struct{ c *massdriver.Client }

func (s sdkAPI) GetS3Credentials(ctx context.Context) (*api.S3Credentials, error) {
	return api.GetS3Credentials(ctx, s.c)
}

// S3Outputs are the formats [RunS3] can write.
var S3Outputs = []string{"process", "env"}

// ValidateS3Output returns an error when output is not one of [S3Outputs].
func ValidateS3Output(output string) error {
	for _, valid := range S3Outputs {
		if output == valid {
			return nil
		}
	}
	return fmt.Errorf("unsupported output format %q: use one of %s", output, strings.Join(S3Outputs, ", "))
}

type processOutput struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
}

// RunS3 gets the S3 credentials and writes them as AWS `credential_process`
// JSON ("process") or as shell exports ("env").
func RunS3(ctx context.Context, client API, output string, out io.Writer) error {
	if err := ValidateS3Output(output); err != nil {
		return err
	}

	creds, err := client.GetS3Credentials(ctx)
	if err != nil {
		return err
	}

	if output == "process" {
		encoded, marshalErr := json.Marshal(processOutput{Version: 1, AccessKeyID: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey})
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal credentials: %w", marshalErr)
		}
		_, err = fmt.Fprintln(out, string(encoded))
		return err
	}

	_, err = fmt.Fprintf(out, "export AWS_ACCESS_KEY_ID=%s\nexport AWS_SECRET_ACCESS_KEY=%s\nexport AWS_ENDPOINT_URL_S3=%s\n",
		shellQuote(creds.AccessKeyID), shellQuote(creds.SecretAccessKey), shellQuote(creds.EndpointURL))
	return err
}

// The env output is run with eval, so each value must stay one literal word.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
