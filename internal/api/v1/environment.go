package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Environment represents a Massdriver deployment environment within a project.
type Environment struct {
	ID          string      `json:"id" mapstructure:"id"`
	Name        string      `json:"name" mapstructure:"name"`
	Description string      `json:"description,omitempty" mapstructure:"description"`
	Cost        CostSummary `json:"cost" mapstructure:"cost"`
	Project     *Project    `json:"project,omitempty" mapstructure:"project,omitempty"`
	Blueprint   *Blueprint  `json:"blueprint,omitempty" mapstructure:"-"`
}

// GetEnvironment retrieves an environment by ID from the Massdriver API.
func GetEnvironment(ctx context.Context, mdClient *client.Client, id string) (*Environment, error) {
	response, err := getEnvironment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", id, err)
	}

	return toEnvironment(response.Environment)
}

// ListEnvironments returns environments, optionally filtered.
func ListEnvironments(ctx context.Context, mdClient *client.Client, filter *EnvironmentsFilter) ([]Environment, error) {
	response, err := listEnvironments(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	envs := make([]Environment, 0, len(response.Environments.Items))
	for _, resp := range response.Environments.Items {
		env, envErr := toEnvironment(resp)
		if envErr != nil {
			return nil, fmt.Errorf("failed to convert environment: %w", envErr)
		}
		envs = append(envs, *env)
	}

	return envs, nil
}

func toEnvironment(v any) (*Environment, error) {
	env := Environment{}
	if err := mapstructure.Decode(v, &env); err != nil {
		return nil, fmt.Errorf("failed to decode environment: %w", err)
	}

	// Unwrap paginated blueprint.instances (API returns {blueprint: {instances: {items: [...]}}})
	type instPage struct {
		Items []Instance `mapstructure:"items"`
	}
	type blueprint struct {
		Instances instPage `mapstructure:"instances"`
	}
	type hasBP struct {
		Blueprint blueprint `mapstructure:"blueprint"`
	}
	var wrapper hasBP
	if err := mapstructure.Decode(v, &wrapper); err == nil && len(wrapper.Blueprint.Instances.Items) > 0 {
		env.Blueprint = &Blueprint{
			Instances: wrapper.Blueprint.Instances.Items,
		}
	}

	return &env, nil
}

// CreateEnvironment creates a new environment within the given project.
func CreateEnvironment(ctx context.Context, mdClient *client.Client, projectID string, input CreateEnvironmentInput) (*Environment, error) {
	response, err := createEnvironment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, projectID, input)
	if err != nil {
		return nil, err
	}
	if !response.CreateEnvironment.Successful {
		messages := response.CreateEnvironment.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to create environment:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to create environment")
	}
	return toEnvironment(response.CreateEnvironment.Result)
}

// UpdateEnvironment updates an environment in the Massdriver API.
func UpdateEnvironment(ctx context.Context, mdClient *client.Client, id string, input UpdateEnvironmentInput) (*Environment, error) {
	response, err := updateEnvironment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.UpdateEnvironment.Successful {
		messages := response.UpdateEnvironment.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to update environment:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to update environment")
	}
	return toEnvironment(response.UpdateEnvironment.Result)
}

// DeleteEnvironment removes an environment by ID from the Massdriver API.
func DeleteEnvironment(ctx context.Context, mdClient *client.Client, id string) (*Environment, error) {
	response, err := deleteEnvironment(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.DeleteEnvironment.Successful {
		messages := response.DeleteEnvironment.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to delete environment:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to delete environment")
	}
	return toEnvironment(response.DeleteEnvironment.Result)
}
