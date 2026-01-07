package pkg

import (
	"context"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Resets a package with options to delete state, params, and deployments.
func RunReset(ctx context.Context, mdClient *client.Client, name string) (*api.Package, error) {
	pkg, err := api.GetPackage(ctx, mdClient, name)

	if err != nil {
		return nil, err
	}

	return api.ResetPackage(ctx, mdClient, pkg.ID)
}
