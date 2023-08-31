package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/debuglog"
)

type Environment struct {
	ID   string
	Slug string
	URL  string
}

const urlTemplate = "https://app.massdriver.cloud/projects/%s/targets/%v"

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectID string, credentials []Credential, packageParams map[string]interface{}, ciContext map[string]interface{}) (*Environment, error) {
	ctx := context.Background()

	input := PreviewEnvironmentInput{
		Credentials:           credentials,
		PackageConfigurations: packageParams,
		CiContext:             ciContext,
	}

	response, err := deployPreviewEnvironment(ctx, client, orgID, projectID, input)

	if err != nil {
		return nil, err
	}

	if response.DeployPreviewEnvironment.Successful {
		return response.DeployPreviewEnvironment.Result.toEnvironment(), nil
	}

	return nil, NewMutationError("failed to deploy environment", response.DeployPreviewEnvironment.Messages)
}

func (e *deployPreviewEnvironmentDeployPreviewEnvironmentEnvironmentPayloadResultEnvironment) toEnvironment() *Environment {
	return &Environment{
		ID:   e.Id,
		Slug: e.Slug,
		// NOTE: We use IDs here instead of slugs because there is currently a bug in the UI for rendering targets w/ slugs.
		URL: fmt.Sprintf(urlTemplate, e.Project.Id, e.Id),
	}
}

func DecommissionPreviewEnvironment(client graphql.Client, orgID string, projectTargetSlugOrTargetID string) (*Environment, error) {
	ctx := context.Background()
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("projectTargetSlugOrTargetID", projectTargetSlugOrTargetID).Logger()
	cmdLog.Info().Msg("Decommissioning preview environment.")

	response, err := decommissionPreviewEnvironment(ctx, client, orgID, projectTargetSlugOrTargetID)

	if err != nil {
		return nil, err
	}

	if response.DecommissionPreviewEnvironment.Successful {
		return response.DecommissionPreviewEnvironment.Result.toEnvironment(), nil
	}

	return nil, NewMutationError("failed to decommission environment", response.DecommissionPreviewEnvironment.Messages)
}

func (e *decommissionPreviewEnvironmentDecommissionPreviewEnvironmentEnvironmentPayloadResultEnvironment) toEnvironment() *Environment {
	return &Environment{
		ID:   e.Id,
		Slug: e.Slug,
		// NOTE: We use IDs here instead of slugs because there is currently a bug in the UI for rendering targets w/ slugs.
		URL: fmt.Sprintf(urlTemplate, e.Project.Id, e.Id),
	}
}
