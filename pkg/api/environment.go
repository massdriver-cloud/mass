package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

const envUrlTemplate = "%s/orgs/%s/projects/%s/environments/%s"

type Environment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Cost        *Cost     `json:"cost,omitempty" mapstructure:"cost,omitempty"`
	Packages    []Package `json:"packages,omitempty" mapstructure:"packages,omitempty"`
	Project     *Project  `json:"project,omitempty" mapstructure:"project,omitempty"`
}

func GetEnvironment(ctx context.Context, mdClient *client.Client, environmentId string) (*Environment, error) {
	response, err := getEnvironmentById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, environmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", environmentId, err)
	}

	return toEnvironment(response.Environment)
}

func GetEnvironmentsByProject(ctx context.Context, mdClient *client.Client, projectId string) ([]Environment, error) {
	response, err := getEnvironmentsByProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectId)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments for project %s: %w", projectId, err)
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

func (e *Environment) URL(mdClient *client.Client) string {
	appURL := strings.Replace(mdClient.Config.URL, "://api.", "://app.", 1)
	return fmt.Sprintf(envUrlTemplate, appURL, mdClient.Config.OrganizationID, e.Project.Slug, e.Slug)
}

func toEnvironment(v any) (*Environment, error) {
	env := Environment{}
	if err := mapstructure.Decode(v, &env); err != nil {
		return nil, fmt.Errorf("failed to decode environment: %w", err)
	}
	return &env, nil
}
