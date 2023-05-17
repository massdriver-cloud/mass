package patch

import (
	"errors"

	"github.com/Khan/genqlient/graphql"
	"github.com/itchyny/gojq"
	"github.com/massdriver-cloud/mass/internal/api"
)

// Updates a packages configuration parameters.
func Run(client graphql.Client, orgID string, name string, setValues []string) (*api.Package, error) {
	pkg, err := api.GetPackageByName(client, orgID, name)

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

			updatedParams, ok = v.(map[string]interface{})

			if !ok {
				return nil, errors.New("failed to cast params")
			}
		}
	}

	return api.ConfigurePackage(client, orgID, pkg.Target.ID, pkg.Manifest.ID, updatedParams)
}
