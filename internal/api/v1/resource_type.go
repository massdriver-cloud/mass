package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// ResourceType defines a category of resource (e.g., "aws-iam-role", "kubernetes-cluster").
// Replaces the v0 concept of "artifact definition".
type ResourceType struct {
	ID                    string         `json:"id" mapstructure:"id"`
	Name                  string         `json:"name" mapstructure:"name"`
	Icon                  string         `json:"icon,omitempty" mapstructure:"icon"`
	ConnectionOrientation string         `json:"connectionOrientation" mapstructure:"connectionOrientation"`
	Schema                map[string]any `json:"schema,omitempty" mapstructure:"schema"`
	CreatedAt             time.Time      `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// GetResourceType retrieves a resource type by ID.
func GetResourceType(ctx context.Context, mdClient *client.Client, id string) (*ResourceType, error) {
	response, err := getResourceType(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource type %s: %w", id, err)
	}
	return toResourceType(response.ResourceType)
}

// ListResourceTypes returns resource types, optionally filtered.
func ListResourceTypes(ctx context.Context, mdClient *client.Client, filter *ResourceTypesFilter) ([]ResourceType, error) {
	var resourceTypes []ResourceType
	var cursor *Cursor

	for {
		response, err := listResourceTypes(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, nil, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource types: %w", err)
		}

		for _, resp := range response.ResourceTypes.Items {
			rt, rtErr := toResourceType(resp)
			if rtErr != nil {
				return nil, fmt.Errorf("failed to convert resource type: %w", rtErr)
			}
			resourceTypes = append(resourceTypes, *rt)
		}

		next := response.ResourceTypes.Cursor.Next
		if next == "" {
			break
		}
		cursor = &Cursor{Next: next}
	}

	return resourceTypes, nil
}

// PublishResourceType upserts a resource type from a JSON Schema document. If an existing
// resource type has the same identifier (from `$md.name` in the schema), its schema is replaced.
//
// Deprecated: transitional shim from V0 `publishArtifactDefinition`. Prefer the OCI-native flow.
func PublishResourceType(ctx context.Context, mdClient *client.Client, input PublishResourceTypeInput) (*ResourceType, error) {
	response, err := publishResourceType(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, input)
	if err != nil {
		return nil, err
	}
	if !response.PublishResourceType.Successful {
		messages := response.PublishResourceType.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to publish resource type:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to publish resource type")
	}
	return toResourceType(response.PublishResourceType.Result)
}

// DeleteResourceType deletes a resource type by ID. Fails if the type is still referenced by
// bundles or existing resources.
//
// Deprecated: transitional shim from V0 `deleteArtifactDefinition`. Prefer the OCI-native flow.
func DeleteResourceType(ctx context.Context, mdClient *client.Client, id string) (*ResourceType, error) {
	response, err := deleteResourceType(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.DeleteResourceType.Successful {
		messages := response.DeleteResourceType.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to delete resource type:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to delete resource type")
	}
	return toResourceType(response.DeleteResourceType.Result)
}

func toResourceType(v any) (*ResourceType, error) {
	rt := ResourceType{}
	if err := mapstructure.Decode(v, &rt); err != nil {
		return nil, fmt.Errorf("failed to decode resource type: %w", err)
	}
	return &rt, nil
}
