package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type Project struct {
	ID            string
	Name          string
	Slug          string
	DefaultParams map[string]interface{}
}

func GetProject(client graphql.Client, orgID string, idOrSlug string) (*Project, error) {
	response, err := getProjectById(context.Background(), client, orgID, idOrSlug)
	p := response.Project.toProject()
	return &p, err
}

func (p *getProjectByIdProject) toProject() Project {
	return Project{
		ID:            p.Id,
		Slug:          p.Slug,
		DefaultParams: p.DefaultParams,
	}
}

func (p *projectsProjectsProject) toProject() Project {
	return Project{
		ID:            p.Id,
		Slug:          p.Slug,
		Name:          p.Name,
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
