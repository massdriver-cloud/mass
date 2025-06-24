package preview

import (
	"context"

	"github.com/massdriver-cloud/mass/pkg/api"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunDecommission decommissions a preview environment
func RunDecommission(ctx context.Context, mdClient *client.Client, projectTargetSlugOrTargetID string) (*api.Environment, error) {
	return api.DecommissionPreviewEnvironment(ctx, mdClient, projectTargetSlugOrTargetID)
}
