// Package credentials prints credentials that other tools use to reach Massdriver.
package credentials

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/internal/api"
)

type processOutput struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
}

// RunS3 writes the credentials as AWS `credential_process` JSON ("process") or
// as shell exports ("env").
func RunS3(creds api.S3Credentials, output string, out io.Writer) error {
	switch output {
	case "process":
		encoded, err := json.Marshal(processOutput{Version: 1, AccessKeyID: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey})
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	case "env":
		fmt.Fprintf(out, "export AWS_ACCESS_KEY_ID=%s\n", creds.AccessKeyID)
		fmt.Fprintf(out, "export AWS_SECRET_ACCESS_KEY=%s\n", creds.SecretAccessKey)
		fmt.Fprintf(out, "export AWS_ENDPOINT_URL_S3=%s\n", creds.EndpointURL)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}
