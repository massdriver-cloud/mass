package deploy

import (
	"encoding/json"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Runs a preview environment deployment
func Run(client graphql.Client, orgID string, projectSlug string, previewCfg *api.PreviewConfig, ciContext *map[string]interface{}) (*api.Environment, error) {
	interpolatedParams, err := interpolateParams(previewCfg.Packages)
	if err != nil {
		return nil, err
	}

	return api.DeployPreviewEnvironment(client, orgID, projectSlug, previewCfg.GetCredentials(), interpolatedParams, *ciContext)
}

func interpolateParams(packages map[string]api.PreviewPackage) (map[string]interface{}, error) {
	interpolatedParams := make(map[string]interface{})
	for id, p := range packages {
		templateData, err := json.Marshal(p.Params)
		if err != nil {
			return nil, err
		}

		config := os.ExpandEnv(string(templateData))

		expanded := make(map[string]interface{})
		err = json.Unmarshal([]byte(config), &expanded)
		if err != nil {
			return nil, err
		}
		interpolatedParams[id] = expanded
	}
	return interpolatedParams, nil
}
