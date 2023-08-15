package deploy

import (
	"encoding/json"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Runs a preview environment deployment
func Run(client graphql.Client, orgID string, projectSlug string, previewCfg *api.PreviewConfig, ciContext *map[string]interface{}) (*api.Environment, error) {
	interpolatedParams := map[string]interface{}{}

	if err := interpolateParams(previewCfg.PackageParams, &interpolatedParams); err != nil {
		return nil, err
	}

	return api.DeployPreviewEnvironment(client, orgID, projectSlug, previewCfg.GetCredentials(), interpolatedParams, *ciContext)
}

func interpolateParams(params map[string]interface{}, interpolatedParams *map[string]interface{}) error {
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
