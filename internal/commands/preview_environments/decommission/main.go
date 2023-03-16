package decommission

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Run decommissions a preview environment
func Run(client graphql.Client, orgID string, projectTargetSlugOrTargetID string) (*api.Environment, error) {
	return api.DecommissionPreviewEnvironment(client, orgID, projectTargetSlugOrTargetID)
}
