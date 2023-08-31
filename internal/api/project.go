package api

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Khan/genqlient/graphql"
)

type Project struct {
	ID            string
	Slug          string
	DefaultParams map[string]interface{}
}

func GetProject(client graphql.Client, orgID string, idOrSlug string) (*Project, error) {
	response, err := getProjectById(context.Background(), client, orgID, idOrSlug)

	return response.Project.toProject(), err
}

func (p *getProjectByIdProject) toProject() *Project {
	return &Project{
		ID:            p.Id,
		Slug:          p.Slug,
		DefaultParams: p.DefaultParams,
	}
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
