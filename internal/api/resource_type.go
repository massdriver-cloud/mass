package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/scalars"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/resourcetypes"
)

// PublishResourceTypeInput is the input for PublishResourceType.
type PublishResourceTypeInput struct {
	Schema map[string]any `json:"schema"`
}

type resourceTypeMutationResult struct {
	Result     *resourcetypes.ResourceType `json:"result"`
	Successful bool                        `json:"successful"`
	Messages   []mutationMessage           `json:"messages"`
}

const publishResourceTypeMutation = `mutation publishResourceType($organizationId: ID!, $input: PublishResourceTypeInput!) {
  publishResourceType(organizationId: $organizationId, input: $input) {
    result {
      id
      name
      version
      icon
      connectionOrientation
      schema
      createdAt
      updatedAt
    }
    successful
    messages {
      code
      field
      message
    }
  }
}`

// PublishResourceType upserts a resource type from a raw JSON Schema document
// via the deprecated mutation, which stores it as the unversioned 0.0.0
// document. No new callers — the massdriver.yaml path publishes through OCI.
func PublishResourceType(ctx context.Context, mdClient *massdriver.Client, input PublishResourceTypeInput) (*resourcetypes.ResourceType, error) {
	cfg := mdClient.Config()

	// The schema field is a GraphQL `Map!` scalar — wire format is a
	// JSON-encoded string. scalars.MarshalJSON is the canonical encoder the
	// genqlient codegen uses; reuse it so the wire shape stays in lockstep.
	schemaRaw, err := scalars.MarshalJSON(input.Schema)
	if err != nil {
		return nil, fmt.Errorf("marshal resource-type schema: %w", err)
	}

	var resp struct {
		PublishResourceType resourceTypeMutationResult `json:"publishResourceType"`
	}
	req := &graphql.Request{
		OpName: "publishResourceType",
		Query:  publishResourceTypeMutation,
		Variables: map[string]any{
			"organizationId": cfg.OrganizationID,
			"input":          map[string]any{"schema": json.RawMessage(schemaRaw)},
		},
	}
	if reqErr := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); reqErr != nil {
		return nil, fmt.Errorf("publish resource type: %w", reqErr)
	}
	if !resp.PublishResourceType.Successful {
		return nil, mutationError("publish resource type", resp.PublishResourceType.Messages)
	}
	if resp.PublishResourceType.Result == nil {
		return nil, errors.New("publish resource type: server reported success but returned no resource type")
	}
	return resp.PublishResourceType.Result, nil
}
