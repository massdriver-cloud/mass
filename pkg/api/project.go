package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Project represents a Massdriver project grouping related environments.
type Project struct {
	ID            string         `json:"id" mapstructure:"id"`
	Name          string         `json:"name" mapstructure:"name"`
	Slug          string         `json:"slug" mapstructure:"slug"`
	Description   string         `json:"description" mapstructure:"description"`
	DefaultParams map[string]any `json:"defaultParams" mapstructure:"defaultParams"`
	Cost          Cost           `json:"cost" mapstructure:"cost"`
	Environments  []Environment  `json:"environments" mapstructure:"environments"`
}

// GetProject retrieves a project by ID or slug from the Massdriver API.
func GetProject(ctx context.Context, mdClient *client.Client, idOrSlug string) (*Project, error) {
	response, err := getProjectById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, idOrSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", idOrSlug, err)
	}

	return toProject(response.Project)
}

func toProject(p any) (*Project, error) {
	proj := Project{}
	if err := mapstructure.Decode(p, &proj); err != nil {
		return nil, fmt.Errorf("failed to decode project: %w", err)
	}
	return &proj, nil
}

// ListProjects returns all projects for the configured organization.
func ListProjects(ctx context.Context, mdClient *client.Client) ([]Project, error) {
	response, err := getProjects(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	records := []Project{}

	for _, resp := range response.Projects {
		proj, projErr := toProject(resp)
		if projErr != nil {
			return nil, fmt.Errorf("failed to convert project: %w", projErr)
		}
		records = append(records, *proj)
	}

	return records, err
}

// GetDefaultParams returns the project's default package parameters as a preview package map.
func (p *Project) GetDefaultParams() map[string]PreviewPackage {
	packages := make(map[string]PreviewPackage)

	for id, prev := range p.DefaultParams {
		var previewPackage PreviewPackage
		jsonPreview, err := json.Marshal(prev)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		err = json.Unmarshal(jsonPreview, &previewPackage.Params)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		packages[id] = previewPackage
	}
	return packages
}

// CreateProject creates a new project in the Massdriver API.
func CreateProject(ctx context.Context, mdClient *client.Client, name string, slug string, description string) (*Project, error) {
	response, err := createProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name, slug, description)
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

// DeleteProject removes a project by ID or slug from the Massdriver API.
func DeleteProject(ctx context.Context, mdClient *client.Client, idOrSlug string) (*Project, error) {
	response, err := deleteProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, idOrSlug)
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
