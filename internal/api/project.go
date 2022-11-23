package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Project struct {
	ID            string
	Slug          string
	DefaultParams map[string]interface{}
}

func GetProject(client graphql.Client, orgID string, idOrSlug string) (Project, error) {
	response, err := getProjectById(context.Background(), client, orgID, idOrSlug)

	return response.Project.toProject(), err
}

func (p *getProjectByIdProject) toProject() Project {
	fmt.Printf("What is %v", p)
	return Project{
		ID:            p.Id,
		Slug:          p.Slug,
		DefaultParams: p.DefaultParams,
	}
}
