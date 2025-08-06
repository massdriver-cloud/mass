package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/debuglog"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

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
		return toEnvironment(response.DeployPreviewEnvironment.Result)
	}

	return nil, NewMutationError("failed to deploy environment", response.DeployPreviewEnvironment.Messages)
}

func DecommissionPreviewEnvironment(ctx context.Context, mdClient *client.Client, projectTargetSlugOrTargetID string) (*Environment, error) {
	cmdLog := debuglog.Log().With().Str("orgID", mdClient.Config.OrganizationID).Str("projectTargetSlugOrTargetID", projectTargetSlugOrTargetID).Logger()
	cmdLog.Info().Msg("Decommissioning preview environment.")

	response, err := decommissionPreviewEnvironment(ctx, mdClient.GQL, mdClient.Config.OrganizationID, projectTargetSlugOrTargetID)

	if err != nil {
		return nil, err
	}

	if response.DecommissionPreviewEnvironment.Successful {
		return toEnvironment(response.DecommissionPreviewEnvironment.Result)
	}

	return nil, NewMutationError("failed to decommission environment", response.DecommissionPreviewEnvironment.Messages)
}
