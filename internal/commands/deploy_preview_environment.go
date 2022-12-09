package commands

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectSlug string) (*api.Environment, error) {
	// TODO: get these values
	credentials := []api.Credential{}
	interpolatedPackageParams := map[string]interface{}{}
	ciContext := map[string]interface{}{}

	return api.DeployPreviewEnvironment(client, orgID, projectSlug, credentials, interpolatedPackageParams, ciContext)
}
