package pkg

import (
	"context"
	"encoding/json"
	"os"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Updates a packages configuration parameters.
func RunConfigure(ctx context.Context, mdClient *client.Client, name string, params map[string]any) (*api.Package, error) {
	pkg, err := api.GetPackageByName(ctx, mdClient, name)

	if err != nil {
		return nil, err
	}

	interpolatedParams := map[string]any{}
	err = interpolateParams(params, &interpolatedParams)

	if err != nil {
		return nil, err
	}

	return api.ConfigurePackage(ctx, mdClient, pkg.Environment.ID, pkg.Manifest.ID, interpolatedParams)
}

func interpolateParams(params map[string]any, interpolatedParams *map[string]any) error {
	templateData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	config := os.ExpandEnv(string(templateData))

	err = json.Unmarshal([]byte(config), &interpolatedParams)

	if err != nil {
		return err
	}

	return nil
}
