package configure

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Updates a packages configuration parameters.
func Run(client graphql.Client, orgID string, name string, params map[string]interface{}) (*api.Package, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)

	if err != nil {
		return nil, err
	}

	interpolatedParams := map[string]interface{}{}
	err = interpolateParams(params, &interpolatedParams)

	if err != nil {
		return nil, err
	}

	return api.ConfigurePackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID, interpolatedParams)
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
