package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Component represents an infrastructure component in a project's blueprint.
type Component struct {
	ID          string `json:"id" mapstructure:"id"`
	Name        string `json:"name" mapstructure:"name"`
	Description string `json:"description,omitempty" mapstructure:"description"`
}

// Link represents a connection between two components in a blueprint.
type Link struct {
	ID        string `json:"id" mapstructure:"id"`
	FromField string `json:"fromField" mapstructure:"fromField"`
	ToField   string `json:"toField" mapstructure:"toField"`
}

// AddComponent adds a component to a project's blueprint.
func AddComponent(ctx context.Context, mdClient *client.Client, projectID string, input AddComponentInput) (*Component, error) {
	response, err := addComponent(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID, input)
	if err != nil {
		return nil, err
	}
	if !response.AddComponent.Successful {
		messages := response.AddComponent.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to add component:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to add component")
	}
	return toComponent(response.AddComponent.Result)
}

// RemoveComponent removes a component from a project's blueprint.
func RemoveComponent(ctx context.Context, mdClient *client.Client, projectID string, id string) (*Component, error) {
	response, err := removeComponent(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID, id)
	if err != nil {
		return nil, err
	}
	if !response.RemoveComponent.Successful {
		messages := response.RemoveComponent.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to remove component:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to remove component")
	}
	return toComponent(response.RemoveComponent.Result)
}

// LinkComponents creates a link between two components in a blueprint.
func LinkComponents(ctx context.Context, mdClient *client.Client, projectID string, input LinkComponentsInput) (*Link, error) {
	response, err := linkComponents(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID, input)
	if err != nil {
		return nil, err
	}
	if !response.LinkComponents.Successful {
		messages := response.LinkComponents.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to link components:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to link components")
	}
	return toLink(response.LinkComponents.Result)
}

// UnlinkComponents removes a link between two components.
func UnlinkComponents(ctx context.Context, mdClient *client.Client, id string) (*Link, error) {
	response, err := unlinkComponents(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.UnlinkComponents.Successful {
		messages := response.UnlinkComponents.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to unlink components:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to unlink components")
	}
	return toLink(response.UnlinkComponents.Result)
}

func toComponent(v any) (*Component, error) {
	comp := Component{}
	if err := mapstructure.Decode(v, &comp); err != nil {
		return nil, fmt.Errorf("failed to decode component: %w", err)
	}
	return &comp, nil
}

func toLink(v any) (*Link, error) {
	link := Link{}
	if err := mapstructure.Decode(v, &link); err != nil {
		return nil, fmt.Errorf("failed to decode link: %w", err)
	}
	return &link, nil
}
