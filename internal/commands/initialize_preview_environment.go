package commands

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func InitializePreviewEnvironment(client graphql.Client, orgID string, projectSlug string) (*PreviewConfig, error) {
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	cfg := PreviewConfig{
		PackageParams: project.DefaultParams,
		// TODO: return Credentials...
	}

	return &cfg, nil
}
