package patch

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Updates a packages configuration parameters.
func Run(client graphql.Client, orgID string, name string, setValues []string) (*api.Package, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)

	updatedParams := pkg.Params

	for _, queryStr := range setValues {
		query, err := gojq.Parse(queryStr)
		if err != nil {
			return nil, err
		}

		iter := query.Run(updatedParams)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return nil, err
			}
			updatedParams = v.(map[string]interface{})
		}
	}

	if err != nil {
		return nil, err
	}

	return api.ConfigurePackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID, updatedParams)
}
