package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Project struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	Description   string         `json:"description"`
	DefaultParams map[string]any `json:"defaultParams"`
	Cost          Cost           `json:"cost"`
	Environments  []Environment  `json:"environments"`
}

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

func ListProjects(ctx context.Context, mdClient *client.Client) ([]Project, error) {
	response, err := getProjects(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	records := []Project{}

	for _, resp := range response.Projects {
		proj, err := toProject(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}
		records = append(records, *proj)
	}

	return records, err
}

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

func CreateProject(ctx context.Context, mdClient *client.Client, name string, slug string, description string) (*Project, error) {
	response, err := createProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name, slug, description)
	if err != nil {
		return nil, err
	}
	if !response.CreateProject.Successful {
		messages := response.CreateProject.GetMessages()
		if len(messages) > 0 {
			errMsg := "unable to create project:"
			for _, msg := range messages {
				errMsg += "\n  - " + msg.Message
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("unable to create project")
	}
	return toProject(response.CreateProject.Result)
}

func DeleteProject(ctx context.Context, mdClient *client.Client, idOrSlug string) (*Project, error) {
	response, err := deleteProject(ctx, mdClient.GQL, mdClient.Config.OrganizationID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if !response.DeleteProject.Successful {
		messages := response.DeleteProject.GetMessages()
		if len(messages) > 0 {
			errMsg := "unable to delete project:"
			for _, msg := range messages {
				errMsg += "\n  - " + msg.Message
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("unable to delete project")
	}
	return toProject(response.DeleteProject.Result)
}
