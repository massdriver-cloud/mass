// Package credentials derives credentials for other tools from a Massdriver access token.
package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	accessTokenPrefix = "md_"
	// The Massdriver S3 API identifies an access token by its display prefix.
	accessKeyIDLength = 12
)

// S3 is the AWS credential set the Massdriver S3 API accepts for one access token.
type S3 struct {
	AccessKeyID     string
	SecretAccessKey string
	EndpointURL     string
}

// NewS3 derives S3 credentials from an access token. The access key ID is the
// token's first 12 characters and the secret access key is the hex SHA-256 of
// the full token.
func NewS3(accessToken, apiURL string) (S3, error) {
	if !strings.HasPrefix(accessToken, accessTokenPrefix) || len(accessToken) < accessKeyIDLength {
		return S3{}, errors.New("S3 credentials need a Massdriver access token (md_...); set one as the profile's API key")
	}

	sum := sha256.Sum256([]byte(accessToken))

	return S3{
		AccessKeyID:     accessToken[:accessKeyIDLength],
		SecretAccessKey: hex.EncodeToString(sum[:]),
		EndpointURL:     strings.TrimRight(apiURL, "/") + "/s3",
	}, nil
}

type processOutput struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
}

// RunS3 writes the credentials as AWS `credential_process` JSON ("process") or
// as shell exports ("env").
func RunS3(creds S3, output string, out io.Writer) error {
	switch output {
	case "process":
		//nolint:gosec // printing the secret is the purpose of this command
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
