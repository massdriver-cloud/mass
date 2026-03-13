package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

const envURLTemplate = "%s/orgs/%s/projects/%s/environments/%s"

// Environment represents a Massdriver deployment environment within a project.
type Environment struct {
	ID          string    `json:"id" mapstructure:"id"`
	Name        string    `json:"name" mapstructure:"name"`
	Slug        string    `json:"slug" mapstructure:"slug"`
	Description string    `json:"description,omitempty" mapstructure:"description"`
	Cost        Cost      `json:"cost" mapstructure:"cost"`
	Packages    []Package `json:"packages,omitempty" mapstructure:"packages,omitempty"`
	Project     *Project  `json:"project,omitempty" mapstructure:"project,omitempty"`
}

// GetEnvironment retrieves an environment by ID from the Massdriver API.
func GetEnvironment(ctx context.Context, mdClient *client.Client, environmentID string) (*Environment, error) {
	response, err := getEnvironmentById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, environmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", environmentID, err)
	}

	return toEnvironment(response.Environment)
}

// GetEnvironmentsByProject retrieves all environments for the given project ID.
func GetEnvironmentsByProject(ctx context.Context, mdClient *client.Client, projectID string) ([]Environment, error) {
	response, err := getEnvironmentsByProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments for project %s: %w", projectID, err)
	}

	envs := make([]Environment, len(response.Project.Environments))
	for idx, resp := range response.Project.Environments {
		env, err := toEnvironment(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert environment: %w", err)
		}
		envs[idx] = *env
	}

	return envs, nil
}

// URL returns the application URL for this environment.
func (e *Environment) URL(ctx context.Context, mdClient *client.Client) string {
	var appURL string
	server, serverErr := GetServer(ctx, mdClient)
	if serverErr != nil {
		// this is greedy (and potentially wrong) but it's VERY unlikely that this query will fail AND the search/replace will be inaccurate
		appURL = strings.Replace(mdClient.Config.URL, "api.", "app.", 1)
	} else {
		appURL = server.AppURL
	}
	return fmt.Sprintf(envURLTemplate, appURL, mdClient.Config.OrganizationID, e.Project.Slug, e.Slug)
}

func toEnvironment(v any) (*Environment, error) {
	env := Environment{}
	if err := mapstructure.Decode(v, &env); err != nil {
		return nil, fmt.Errorf("failed to decode environment: %w", err)
	}
	return &env, nil
}

// CreateEnvironment creates a new environment within the given project.
func CreateEnvironment(ctx context.Context, mdClient *client.Client, projectID string, name string, slug string, description string) (*Environment, error) {
	response, err := createEnvironment(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID, name, slug, description)
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

// SetEnvironmentDefault sets the default artifact connection for an environment.
func SetEnvironmentDefault(ctx context.Context, mdClient *client.Client, environmentID string, artifactID string) error {
	response, err := createEnvironmentConnection(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID, environmentID)
	if err != nil {
		return fmt.Errorf("failed to set environment default: %w", err)
	}
	if !response.CreateEnvironmentConnection.Successful {
		messages := response.CreateEnvironmentConnection.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to set environment default:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return errors.New(sb.String())
		}
		return errors.New("unable to set environment default")
	}
	return nil
}
