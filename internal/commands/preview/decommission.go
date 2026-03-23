// Package preview provides commands for managing preview environments in Massdriver.
package preview

import (
	"context"

	"github.com/massdriver-cloud/mass/internal/api/v0"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunDecommission decommissions a preview environment
func RunDecommission(ctx context.Context, mdClient *client.Client, projectTargetSlugOrTargetID string) (*api.Environment, error) {
	return api.DecommissionPreviewEnvironment(ctx, mdClient, projectTargetSlugOrTargetID)
}
