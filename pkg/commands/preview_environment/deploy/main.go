package deploy

import (
	"encoding/json"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/api"
)

// Runs a preview environment deployment
func Run(client graphql.Client, orgID string, projectSlug string, previewCfg *api.PreviewConfig, ciContext *map[string]interface{}) (*api.Environment, error) {
	packagesWithInterpolatedParams, err := interpolateParams(previewCfg.Packages)

	if err != nil {
		return nil, err
	}

	return api.DeployPreviewEnvironment(client, orgID, projectSlug, previewCfg.GetCredentials(), packagesWithInterpolatedParams, *ciContext)
}

func interpolateParams(packages map[string]api.PreviewPackage) (map[string]api.PreviewPackage, error) {
	for slug, p := range packages {
		templateData, err := json.Marshal(p.Params)
		if err != nil {
			return nil, err
		}

		config := os.ExpandEnv(string(templateData))

		expandedParams := make(map[string]interface{})
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
