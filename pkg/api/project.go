package api

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Project struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Slug               string                 `json:"slug"`
	Description        string                 `json:"description"`
	DefaultParams      map[string]interface{} `json:"defaultParams"`
	MonthlyAverageCost float64                `json:"monthlyAverageCost"`
	DailyAverageCost   float64                `json:"dailyAverageCost"`
	Environments       []Environment          `json:"environments"`
}

func GetProject(ctx context.Context, mdClient *client.Client, idOrSlug string) (*Project, error) {
	response, err := getProjectById(ctx, mdClient.GQL, mdClient.Config.OrganizationID, idOrSlug)

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

func (p *getProjectsProjectsProject) toProject() Project {
	environments := make([]Environment, len(p.Environments))
	for i, env := range p.Environments {
		environments[i] = Environment{
			Name: env.Name,
			Slug: env.Slug,
		}
	}

	return Project{
		ID:                 p.Id,
		Slug:               p.Slug,
		Name:               p.Name,
		Description:        p.Description,
		DefaultParams:      p.DefaultParams,
		MonthlyAverageCost: p.Cost.Monthly.Average.Amount,
		DailyAverageCost:   p.Cost.Daily.Average.Amount,
		Environments:       environments,
	}
}

func ListProjects(ctx context.Context, mdClient *client.Client) ([]Project, error) {
	response, err := getProjects(ctx, mdClient.GQL, mdClient.Config.OrganizationID)
	records := []Project{}

	for _, prj := range response.Projects {
		records = append(records, prj.toProject())
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
