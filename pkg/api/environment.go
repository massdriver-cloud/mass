package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/debuglog"
)

type Environment struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

const urlTemplate = "https://app.massdriver.cloud/orgs/%s/projects/%s/targets/%v"

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectID string, credentials []Credential, packageParams map[string]PreviewPackage, ciContext map[string]interface{}) (*Environment, error) {
	// Validate that no package has both params and remote references
	for packageName, pkg := range packageParams {
		if pkg.Params != nil && len(pkg.RemoteReferences) > 0 {
			return nil, fmt.Errorf("package '%s': \"params\" and \"remoteReferences\" are mutually exclusive", packageName)
		}
	}

	ctx := context.Background()

	packageParamsJSON := make(map[string]interface{})
	for k, v := range packageParams {
		packageParamsJSON[k] = v
	}

	input := PreviewEnvironmentInput{
		Credentials:           credentials,
		PackageConfigurations: packageParamsJSON,
		CiContext:             ciContext,
	}

	response, err := deployPreviewEnvironment(ctx, client, orgID, projectID, input)

	if err != nil {
		return nil, err
	}

	if response.DeployPreviewEnvironment.Successful {
		return response.DeployPreviewEnvironment.Result.toEnvironment(orgID), nil
	}

	return nil, NewMutationError("failed to deploy environment", response.DeployPreviewEnvironment.Messages)
}

func (e *deployPreviewEnvironmentDeployPreviewEnvironmentEnvironmentPayloadResultEnvironment) toEnvironment(orgID string) *Environment {
	return &Environment{
		ID:   e.Id,
		Slug: e.Slug,

		// NOTE: We use IDs here instead of slugs because there is currently a bug in the UI for rendering targets w/ slugs.
		URL: fmt.Sprintf(urlTemplate, orgID, e.Project.Id, e.Id),
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
		return response.DecommissionPreviewEnvironment.Result.toEnvironment(orgID), nil
	}

	return nil, NewMutationError("failed to decommission environment", response.DecommissionPreviewEnvironment.Messages)
}

func (e *decommissionPreviewEnvironmentDecommissionPreviewEnvironmentEnvironmentPayloadResultEnvironment) toEnvironment(orgID string) *Environment {
	return &Environment{
		ID:   e.Id,
		Slug: e.Slug,
		// NOTE: We use IDs here instead of slugs because there is currently a bug in the UI for rendering targets w/ slugs.
		URL: fmt.Sprintf(urlTemplate, orgID, e.Project.Id, e.Id),
	}
}
