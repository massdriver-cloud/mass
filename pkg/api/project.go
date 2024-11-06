package api

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Khan/genqlient/graphql"
)

type Project struct {
	ID            string
	Name          string
	Slug          string
	Description   string
	DefaultParams map[string]interface{}
}

func GetProject(client graphql.Client, orgID string, idOrSlug string) (*Project, error) {
	response, err := getProjectById(context.Background(), client, orgID, idOrSlug)

	return response.Project.toProject(), err
}

func (p *getProjectByIdProject) toProject() *Project {
	return &Project{
		ID:            p.Id,
		Name:          p.Name,
		Slug:          p.Slug,
		Description:   p.Description,
		DefaultParams: p.DefaultParams,
	}
}

func (p *projectsProjectsProject) toProject() Project {
	return Project{
		ID:            p.Id,
		Slug:          p.Slug,
		Name:          p.Name,
		Description:   p.Description,
		DefaultParams: p.DefaultParams,
	}
}

func ListProjects(client graphql.Client, orgID string) (*[]Project, error) {
	response, err := projects(context.Background(), client, orgID)
	records := []Project{}

	for _, prj := range response.Projects {
		records = append(records, prj.toProject())
	}

	return &records, err
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
