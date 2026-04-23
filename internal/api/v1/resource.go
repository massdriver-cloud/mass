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

// Resource is an infrastructure artifact such as cloud credentials, a database connection string,
// or any other output produced by (or imported into) Massdriver.
// Replaces the v0 concept of "artifact".
type Resource struct {
	ID           string         `json:"id" mapstructure:"id"`
	Name         string         `json:"name" mapstructure:"name"`
	Origin       string         `json:"origin" mapstructure:"origin"`
	ResourceType *ResourceType  `json:"resourceType,omitempty" mapstructure:"resourceType,omitempty"`
	Field        string         `json:"field,omitempty" mapstructure:"field"`
	Instance     *Instance      `json:"instance,omitempty" mapstructure:"instance,omitempty"`
	Formats      []string       `json:"formats,omitempty" mapstructure:"formats"`
	Payload      map[string]any `json:"payload,omitempty" mapstructure:"payload,omitempty"`
	CreatedAt    time.Time      `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// GetResource retrieves a resource by ID.
func GetResource(ctx context.Context, mdClient *client.Client, id string) (*Resource, error) {
	response, err := getResource(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", id, err)
	}
	return toResource(response.Resource)
}

// ListResources returns resources, optionally filtered.
func ListResources(ctx context.Context, mdClient *client.Client, filter *ResourcesFilter) ([]Resource, error) {
	var resources []Resource
	var cursor *Cursor

	for {
		response, err := listResources(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, nil, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}

		for _, resp := range response.Resources.Items {
			r, rErr := toResource(resp)
			if rErr != nil {
				return nil, fmt.Errorf("failed to convert resource: %w", rErr)
			}
			resources = append(resources, *r)
		}

		next := response.Resources.Cursor.Next
		if next == "" {
			break
		}
		cursor = &Cursor{Next: next}
	}

	return resources, nil
}

// CreateResource imports a new resource of the given type.
func CreateResource(ctx context.Context, mdClient *client.Client, resourceTypeID string, input CreateResourceInput) (*Resource, error) {
	response, err := createResource(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, resourceTypeID, input)
	if err != nil {
		return nil, err
	}
	if !response.CreateResource.Successful {
		messages := response.CreateResource.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to create resource:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to create resource")
	}
	return toResource(response.CreateResource.Result)
}

// UpdateResource updates an existing resource.
func UpdateResource(ctx context.Context, mdClient *client.Client, id string, input UpdateResourceInput) (*Resource, error) {
	response, err := updateResource(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.UpdateResource.Successful {
		messages := response.UpdateResource.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to update resource:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to update resource")
	}
	return toResource(response.UpdateResource.Result)
}

// ResourceWithSensitiveValues is a resource whose `$md.sensitive` payload fields are unmasked.
// Returned by ExportResource; requesting it is recorded in the audit log.
type ResourceWithSensitiveValues struct {
	ID           string         `json:"id" mapstructure:"id"`
	Name         string         `json:"name" mapstructure:"name"`
	Origin       string         `json:"origin" mapstructure:"origin"`
	ResourceType *ResourceType  `json:"resourceType,omitempty" mapstructure:"resourceType,omitempty"`
	Payload      map[string]any `json:"payload,omitempty" mapstructure:"payload,omitempty"`
	Rendered     string         `json:"rendered" mapstructure:"rendered"`
	CreatedAt    time.Time      `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// ExportResource returns a resource with sensitive payload fields unmasked, along with a `rendered`
// string in the requested format. An empty format defaults to the API's default (json).
func ExportResource(ctx context.Context, mdClient *client.Client, id, format string) (*ResourceWithSensitiveValues, error) {
	response, err := exportResource(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, format)
	if err != nil {
		return nil, err
	}
	if !response.ExportResource.Successful {
		messages := response.ExportResource.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to export resource:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to export resource")
	}
	return toResourceWithSensitiveValues(response.ExportResource.Result)
}

// DeleteResource deletes an imported resource by ID.
func DeleteResource(ctx context.Context, mdClient *client.Client, id string) (*Resource, error) {
	response, err := deleteResource(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.DeleteResource.Successful {
		messages := response.DeleteResource.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to delete resource:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to delete resource")
	}
	return toResource(response.DeleteResource.Result)
}

func toResource(v any) (*Resource, error) {
	r := Resource{}
	if err := mapstructure.Decode(v, &r); err != nil {
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}
	return &r, nil
}

func toResourceWithSensitiveValues(v any) (*ResourceWithSensitiveValues, error) {
	r := ResourceWithSensitiveValues{}
	if err := mapstructure.Decode(v, &r); err != nil {
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}
	return &r, nil
}
