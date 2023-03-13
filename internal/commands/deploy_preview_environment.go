package commands

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectSlug string, previewCfg *api.PreviewConfig, ciContext *map[string]interface{}) (*api.Environment, error) {
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

	envVars := getOsEnv()
	config := os.Expand(string(templateData), func(s string) string { return envVars[s] })

	if err = json.Unmarshal([]byte(config), &interpolatedParams); err != nil {
		return err
	}

	return nil
}

func getOsEnv() map[string]string {
	getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getkeyval(item)
			items[key] = val
		}
		return items
	}

	osEnv := getenvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})

	return osEnv
}
