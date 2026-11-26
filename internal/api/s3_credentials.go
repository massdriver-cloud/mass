package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// S3Credentials are the AWS credentials of the Massdriver S3 API for the
// access token that requested them.
type S3Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	EndpointURL     string `json:"endpointUrl"`
}

//nolint:gosec // a GraphQL query that names the secret field, not a credential
const s3CredentialsQuery = `query s3Credentials($organizationId: ID!) {
  s3Credentials(organizationId: $organizationId) {
    accessKeyId
    secretAccessKey
    endpointUrl
  }
}`

// GetS3Credentials gets the S3 API credentials of the access token that the
// client authenticates with.
func GetS3Credentials(ctx context.Context, mdClient *massdriver.Client) (*S3Credentials, error) {
	var resp struct {
		S3Credentials S3Credentials `json:"s3Credentials"`
	}
	req := &graphql.Request{
		OpName:    "s3Credentials",
		Query:     s3CredentialsQuery,
		Variables: map[string]any{"organizationId": mdClient.Config().OrganizationID},
	}
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, fmt.Errorf("get S3 credentials: %w", err)
	}
	return &resp.S3Credentials, nil
}
