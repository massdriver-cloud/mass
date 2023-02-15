package commands

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func InitializePreviewEnvironment(client graphql.Client, orgID string, projectSlug string) (*PreviewConfig, error) {
	// TODO: Take stdin & prompt w/ bubbletea
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	// selectedArtifactTypes, _ := credential_types_table.New(api.ListCredentialTypes())

	// fmt.Printf("What is %v", selectedArtifactTypes)

	cfg := PreviewConfig{
		PackageParams: project.DefaultParams,
		// TODO: return Credentials...
	}

	return &cfg, nil
}
