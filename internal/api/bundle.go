package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
)

// BundleDependencyCardinalities is a published bundle version and the
// cardinality (ONE or MANY) of each of its dependencies, keyed by name.
type BundleDependencyCardinalities struct {
	Version       string
	Cardinalities map[string]string
}

const getBundleDependencyCardinalitiesQuery = `query getBundleDependencyCardinalities($organizationId: ID!, $id: BundleId!) {
  bundle(organizationId: $organizationId, id: $id) {
    version
    dependencies {
      name
      cardinality
    }
  }
}`

// GetBundleDependencyCardinalities fetches the dependency cardinalities of the
// bundle that id resolves to. Returns [gql.ErrNotFound] (wrapped) when no
// version matches.
func GetBundleDependencyCardinalities(ctx context.Context, mdClient *massdriver.Client, id string) (*BundleDependencyCardinalities, error) {
	var resp struct {
		Bundle *struct {
			Version      string `json:"version"`
			Dependencies []struct {
				Name        string `json:"name"`
				Cardinality string `json:"cardinality"`
			} `json:"dependencies"`
		} `json:"bundle"`
	}
	req := &graphql.Request{
		OpName: "getBundleDependencyCardinalities",
		Query:  getBundleDependencyCardinalitiesQuery,
		Variables: map[string]any{
			"organizationId": mdClient.Config().OrganizationID,
			"id":             id,
		},
	}
	if err := gqlClient(mdClient).MakeRequest(ctx, req, &graphql.Response{Data: &resp}); err != nil {
		return nil, gql.ClassifyError(fmt.Errorf("get bundle %s: %w", id, err))
	}
	if resp.Bundle == nil {
		return nil, fmt.Errorf("get bundle %s: %w", id, gql.ErrNotFound)
	}

	result := &BundleDependencyCardinalities{
		Version:       resp.Bundle.Version,
		Cardinalities: make(map[string]string, len(resp.Bundle.Dependencies)),
	}
	for _, dep := range resp.Bundle.Dependencies {
		result.Cardinalities[dep.Name] = dep.Cardinality
	}
	return result, nil
}
