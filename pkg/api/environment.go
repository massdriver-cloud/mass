package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/debuglog"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Environment struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug"`
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

const urlTemplate = "https://app.massdriver.cloud/orgs/%s/projects/%s/targets/%v"

func DeployPreviewEnvironment(ctx context.Context, mdClient *client.Client, projectID string, credentials []Credential, packageParams map[string]PreviewPackage, ciContext map[string]any) (*Environment, error) {
	// Validate that no package has both params and remote references
	for packageName, pkg := range packageParams {
		if pkg.Params != nil && len(pkg.RemoteReferences) > 0 {
			return nil, fmt.Errorf("package '%s': \"params\" and \"remoteReferences\" are mutually exclusive", packageName)
		}
	}

	packageParamsJSON := make(map[string]any)
	for k, v := range packageParams {
		packageParamsJSON[k] = v
	}

	input := PreviewEnvironmentInput{
		Credentials:           credentials,
		PackageConfigurations: packageParamsJSON,
		CiContext:             ciContext,
	}

	response, err := deployPreviewEnvironment(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectID, input)

	if err != nil {
		return nil, err
	}

	if response.DeployPreviewEnvironment.Successful {
		return response.DeployPreviewEnvironment.Result.toEnvironment(mdClient.Config.OrganizationID), nil
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

func DecommissionPreviewEnvironment(ctx context.Context, mdClient *client.Client, projectTargetSlugOrTargetID string) (*Environment, error) {
	cmdLog := debuglog.Log().With().Str("orgID", mdClient.Config.OrganizationID).Str("projectTargetSlugOrTargetID", projectTargetSlugOrTargetID).Logger()
	cmdLog.Info().Msg("Decommissioning preview environment.")

	response, err := decommissionPreviewEnvironment(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectTargetSlugOrTargetID)

	if err != nil {
		return nil, err
	}

	if response.DecommissionPreviewEnvironment.Successful {
		return response.DecommissionPreviewEnvironment.Result.toEnvironment(mdClient.Config.OrganizationID), nil
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
