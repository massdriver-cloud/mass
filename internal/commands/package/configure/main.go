package configure

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Updates a packages configuration parameters.
func Run(client graphql.Client, orgID string, name string, params map[string]interface{}) (*api.Package, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)

	if err != nil {
		return nil, err
	}

	return api.ConfigurePackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID, params)
}
