package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Project represents a Massdriver project.
type Project struct {
	ID           string            `json:"id" mapstructure:"id"`
	Name         string            `json:"name" mapstructure:"name"`
	Description  string            `json:"description" mapstructure:"description"`
	Cost         CostSummary       `json:"cost" mapstructure:"cost"`
	Attributes   map[string]string `json:"attributes,omitempty" mapstructure:"attributes"`
	Environments []Environment     `json:"environments,omitempty" mapstructure:"-"`
}

// GetProject retrieves a project by ID from the Massdriver API.
func GetProject(ctx context.Context, mdClient *client.Client, id string) (*Project, error) {
	response, err := getProject(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", id, err)
	}

	return toProject(response.Project)
}

// ListProjects returns all projects for the configured organization.
func ListProjects(ctx context.Context, mdClient *client.Client) ([]Project, error) {
	response, err := listProjects(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	records := make([]Project, 0, len(response.Projects.Items))
	for _, resp := range response.Projects.Items {
		proj, projErr := toProject(resp)
		if projErr != nil {
			return nil, fmt.Errorf("failed to convert project: %w", projErr)
		}
		records = append(records, *proj)
	}

	return records, nil
}

func toProject(p any) (*Project, error) {
	proj := Project{}
	if err := decode(p, &proj); err != nil {
		return nil, fmt.Errorf("failed to decode project: %w", err)
	}

	// Unwrap paginated environments (API returns {items: [...]})
	type envPage struct {
		Items []Environment `json:"items"`
	}
	type hasEnvs struct {
		Environments envPage `json:"environments"`
	}
	var wrapper hasEnvs
	if err := decode(p, &wrapper); err == nil && len(wrapper.Environments.Items) > 0 {
		proj.Environments = wrapper.Environments.Items
	}

	return &proj, nil
}

// CreateProject creates a new project in the Massdriver API.
func CreateProject(ctx context.Context, mdClient *client.Client, input CreateProjectInput) (*Project, error) {
	response, err := createProject(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, input)
	if err != nil {
		return nil, err
	}
	if !response.CreateProject.Successful {
		messages := response.CreateProject.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to create project:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to create project")
	}
	return toProject(response.CreateProject.Result)
}

// UpdateProject updates a project in the Massdriver API.
func UpdateProject(ctx context.Context, mdClient *client.Client, id string, input UpdateProjectInput) (*Project, error) {
	response, err := updateProject(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.UpdateProject.Successful {
		messages := response.UpdateProject.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to update project:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to update project")
	}
	return toProject(response.UpdateProject.Result)
}

// DeleteProject removes a project by ID from the Massdriver API.
func DeleteProject(ctx context.Context, mdClient *client.Client, id string) (*Project, error) {
	response, err := deleteProject(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.DeleteProject.Successful {
		messages := response.DeleteProject.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to delete project:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to delete project")
	}
	return toProject(response.DeleteProject.Result)
}
