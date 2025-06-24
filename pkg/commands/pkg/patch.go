package pkg

import (
	"context"
	"errors"

	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Updates a packages configuration parameters.
func RunPatch(ctx context.Context, mdClient *client.Client, name string, setValues []string) (*api.Package, error) {
	pkg, err := api.GetPackageByName(ctx, mdClient, name)

	if err != nil {
		return nil, err
	}

	updatedParams := pkg.Params

	for _, queryStr := range setValues {
		query, parseErr := gojq.Parse(queryStr)

		if parseErr != nil {
			return nil, parseErr
		}

		iter := query.Run(updatedParams)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok = v.(error); ok {
				return nil, err
			}

			updatedParams, ok = v.(map[string]any)

			if !ok {
				return nil, errors.New("failed to cast params")
			}
		}
	}

	return api.ConfigurePackage(ctx, mdClient, pkg.Environment.ID, pkg.Manifest.ID, updatedParams)
}
