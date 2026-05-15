package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/scalars"
)

// ResourceType mirrors the v2 GraphQL schema's resource-type record. Field
// names match the JSON wire shape so handcrafted GraphQL responses decode
// without bespoke mapping.
type ResourceType struct {
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	Icon                  string         `json:"icon,omitempty"`
	ConnectionOrientation string         `json:"connectionOrientation"`
	Schema                map[string]any `json:"schema,omitempty"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

// PublishResourceTypeInput is the input for PublishResourceType.
type PublishResourceTypeInput struct {
	Schema map[string]any `json:"schema"`
}

// resourceTypeMutationResult is the wrapped payload every resource-type
// mutation returns.
type resourceTypeMutationResult struct {
	Result     *ResourceType     `json:"result"`
	Successful bool              `json:"successful"`
	Messages   []mutationMessage `json:"messages"`
}

const getResourceTypeQuery = `query getResourceType($organizationId: ID!, $id: ID!) {
  resourceType(organizationId: $organizationId, id: $id) {
    id
    name
    icon
    connectionOrientation
    schema
    createdAt
    updatedAt
  }
}`

const listResourceTypesQuery = `query listResourceTypes($organizationId: ID!) {
  resourceTypes(organizationId: $organizationId) {
    items {
      id
      name
      icon
      connectionOrientation
      createdAt
      updatedAt
    }
  }
}`

const publishResourceTypeMutation = `mutation publishResourceType($organizationId: ID!, $input: PublishResourceTypeInput!) {
  publishResourceType(organizationId: $organizationId, input: $input) {
    result {
      id
      name
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

const deleteResourceTypeMutation = `mutation deleteResourceType($organizationId: ID!, $id: ID!) {
  deleteResourceType(organizationId: $organizationId, id: $id) {
    result {
      id
      name
    }
    successful
    messages {
      code
      field
      message
    }
  }
}`

// GetResourceType fetches a single resource type by name.
func GetResourceType(ctx context.Context, mdClient *massdriver.Client, name string) (*ResourceType, error) {
	cfg := mdClient.Config()
	var resp struct {
		ResourceType *ResourceType `json:"resourceType"`
	}
	req := &graphql.Request{
		OpName: "getResourceType",
		Query:  getResourceTypeQuery,
		Variables: map[string]any{
			"organizationId": cfg.OrganizationID,
			"id":             name,
		},
	}
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, fmt.Errorf("get resource type %s: %w", name, err)
	}
	if resp.ResourceType == nil {
		return nil, fmt.Errorf("get resource type %s: %w", name, gql.ErrNotFound)
	}
	return resp.ResourceType, nil
}

// ListResourceTypes fetches all resource types in the configured organization.
// The legacy CLI supported a filter argument; the few callsites that survive
// the v2 migration only need the unfiltered list.
func ListResourceTypes(ctx context.Context, mdClient *massdriver.Client) ([]ResourceType, error) {
	cfg := mdClient.Config()
	var resp struct {
		ResourceTypes struct {
			Items []ResourceType `json:"items"`
		} `json:"resourceTypes"`
	}
	req := &graphql.Request{
		OpName: "listResourceTypes",
		Query:  listResourceTypesQuery,
		Variables: map[string]any{
			"organizationId": cfg.OrganizationID,
		},
	}
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, fmt.Errorf("list resource types: %w", err)
	}
	return resp.ResourceTypes.Items, nil
}

// PublishResourceType registers a resource-type schema.
func PublishResourceType(ctx context.Context, mdClient *massdriver.Client, input PublishResourceTypeInput) (*ResourceType, error) {
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
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, fmt.Errorf("publish resource type: %w", err)
	}
	if !resp.PublishResourceType.Successful {
		return nil, mutationError("publish resource type", resp.PublishResourceType.Messages)
	}
	return resp.PublishResourceType.Result, nil
}

// DeleteResourceType removes a resource type by name.
func DeleteResourceType(ctx context.Context, mdClient *massdriver.Client, name string) (*ResourceType, error) {
	cfg := mdClient.Config()
	var resp struct {
		DeleteResourceType resourceTypeMutationResult `json:"deleteResourceType"`
	}
	req := &graphql.Request{
		OpName: "deleteResourceType",
		Query:  deleteResourceTypeMutation,
		Variables: map[string]any{
			"organizationId": cfg.OrganizationID,
			"id":             name,
		},
	}
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, fmt.Errorf("delete resource type %s: %w", name, err)
	}
	if !resp.DeleteResourceType.Successful {
		return nil, mutationError("delete resource type "+name, resp.DeleteResourceType.Messages)
	}
	return resp.DeleteResourceType.Result, nil
}
