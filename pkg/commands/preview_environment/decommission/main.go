package decommission

import (
	"context"

	"github.com/massdriver-cloud/mass/pkg/api"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Run decommissions a preview environment
func Run(ctx context.Context, mdClient *client.Client, projectTargetSlugOrTargetID string) (*api.Environment, error) {
	return api.DecommissionPreviewEnvironment(ctx, mdClient, projectTargetSlugOrTargetID)
}
