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

// Component represents a slot in a project's blueprint backed by a bundle.
type Component struct {
	ID          string            `json:"id" mapstructure:"id"`
	Name        string            `json:"name" mapstructure:"name"`
	Description string            `json:"description,omitempty" mapstructure:"description"`
	Tags        map[string]string `json:"tags,omitempty" mapstructure:"tags"`
	OciRepo     *OciRepo          `json:"ociRepo,omitempty" mapstructure:"ociRepo,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// Link represents a design-time wire between two components in a blueprint.
type Link struct {
	ID            string     `json:"id" mapstructure:"id"`
	FromField     string     `json:"fromField" mapstructure:"fromField"`
	ToField       string     `json:"toField" mapstructure:"toField"`
	FromComponent *Component `json:"fromComponent,omitempty" mapstructure:"fromComponent,omitempty"`
	ToComponent   *Component `json:"toComponent,omitempty" mapstructure:"toComponent,omitempty"`
	CreatedAt     time.Time  `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// ListLinks returns every link in a project's blueprint, optionally filtered, following pagination.
func ListLinks(ctx context.Context, mdClient *client.Client, projectID string, filter *LinksFilter) ([]Link, error) {
	var links []Link
	var cursor *Cursor

	for {
		response, err := listLinks(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, projectID, filter, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list links for project %s: %w", projectID, err)
		}

		for _, item := range response.Project.Blueprint.Links.Items {
			l, decodeErr := toLink(item)
			if decodeErr != nil {
				return nil, fmt.Errorf("failed to convert link: %w", decodeErr)
			}
			links = append(links, *l)
		}

		next := response.Project.Blueprint.Links.Cursor.Next
		if next == "" {
			break
		}
		cursor = &Cursor{Next: next}
	}

	return links, nil
}

// AddComponent adds a component to a project's blueprint.
func AddComponent(ctx context.Context, mdClient *client.Client, projectID, ociRepoName string, input AddComponentInput) (*Component, error) {
	response, err := addComponent(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, projectID, ociRepoName, input)
	if err != nil {
		return nil, err
	}
	if !response.AddComponent.Successful {
		messages := make([]string, 0, len(response.AddComponent.Messages))
		for _, m := range response.AddComponent.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to add component", messages)
	}
	return toComponent(response.AddComponent.Result)
}

// RemoveComponent removes a component from a project's blueprint.
func RemoveComponent(ctx context.Context, mdClient *client.Client, componentID string) (*Component, error) {
	response, err := removeComponent(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, componentID)
	if err != nil {
		return nil, err
	}
	if !response.RemoveComponent.Successful {
		messages := make([]string, 0, len(response.RemoveComponent.Messages))
		for _, m := range response.RemoveComponent.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to remove component", messages)
	}
	return toComponent(response.RemoveComponent.Result)
}

// LinkComponents creates a design-time link between two components.
func LinkComponents(ctx context.Context, mdClient *client.Client, input LinkComponentsInput) (*Link, error) {
	response, err := linkComponents(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, input)
	if err != nil {
		return nil, err
	}
	if !response.LinkComponents.Successful {
		messages := make([]string, 0, len(response.LinkComponents.Messages))
		for _, m := range response.LinkComponents.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to link components", messages)
	}
	return toLink(response.LinkComponents.Result)
}

// UnlinkComponents removes a link by its ID.
func UnlinkComponents(ctx context.Context, mdClient *client.Client, linkID string) (*Link, error) {
	response, err := unlinkComponents(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, linkID)
	if err != nil {
		return nil, err
	}
	if !response.UnlinkComponents.Successful {
		messages := make([]string, 0, len(response.UnlinkComponents.Messages))
		for _, m := range response.UnlinkComponents.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to unlink components", messages)
	}
	return toLink(response.UnlinkComponents.Result)
}

func toComponent(v any) (*Component, error) {
	c := Component{}
	if err := mapstructure.Decode(v, &c); err != nil {
		return nil, fmt.Errorf("failed to decode component: %w", err)
	}
	return &c, nil
}

func toLink(v any) (*Link, error) {
	l := Link{}
	if err := mapstructure.Decode(v, &l); err != nil {
		return nil, fmt.Errorf("failed to decode link: %w", err)
	}
	return &l, nil
}

func mutationError(prefix string, messages []string) error {
	if len(messages) == 0 {
		return errors.New(prefix)
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(":")
	for _, msg := range messages {
		sb.WriteString("\n  - ")
		sb.WriteString(msg)
	}
	return errors.New(sb.String())
}
