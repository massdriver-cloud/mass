package instance

import (
	"context"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunReset resets a package state to 'Initialized'.
func RunReset(ctx context.Context, mdClient *client.Client, name string) (*api.Package, error) {
	pkg, err := api.GetPackage(ctx, mdClient, name)

	if err != nil {
		return nil, err
	}

	return api.ResetPackage(ctx, mdClient, pkg.ID)
}
