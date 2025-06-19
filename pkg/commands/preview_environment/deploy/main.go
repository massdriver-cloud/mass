package deploy

import (
	"context"
	"encoding/json"
	"os"

	"github.com/massdriver-cloud/mass/pkg/api"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Runs a preview environment deployment
func Run(ctx context.Context, mdClient *client.Client, projectSlug string, previewCfg *api.PreviewConfig, ciContext *map[string]any) (*api.Environment, error) {
	packagesWithInterpolatedParams, err := interpolateParams(previewCfg.Packages)

	if err != nil {
		return nil, err
	}

	return api.DeployPreviewEnvironment(ctx, mdClient, projectSlug, previewCfg.GetCredentials(), packagesWithInterpolatedParams, *ciContext)
}

func interpolateParams(packages map[string]api.PreviewPackage) (map[string]api.PreviewPackage, error) {
	for slug, p := range packages {
		templateData, err := json.Marshal(p.Params)
		if err != nil {
			return nil, err
		}

		config := os.ExpandEnv(string(templateData))

		expandedParams := make(map[string]any)
		err = json.Unmarshal([]byte(config), &expandedParams)
		if err != nil {
			return nil, err
		}

		pkg := packages[slug]
		pkg.Params = expandedParams

		packages[slug] = pkg
	}

	return packages, nil
}
