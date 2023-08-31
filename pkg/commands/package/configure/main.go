package configure

import (
	"encoding/json"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/api"
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

	config := os.ExpandEnv(string(templateData))

	err = json.Unmarshal([]byte(config), &interpolatedParams)

	if err != nil {
		return err
	}

	return nil
}
