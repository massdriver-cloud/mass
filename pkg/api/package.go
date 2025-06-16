package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Package struct {
	ID          string                 `json:"id"`
	NamePrefix  string                 `json:"namePrefix"`
	Params      map[string]interface{} `json:"params"`
	Manifest    Manifest               `json:"manifest"`
	Environment Environment            `json:"environment"`
}

func (p *Package) ParamsJSON() (string, error) {
	paramsJSON, err := json.MarshalIndent(p.Params, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal params to JSON: %w", err)
	}
	return string(paramsJSON), nil
}

func GetPackageByName(client graphql.Client, orgID string, name string) (*Package, error) {
	ctx := context.Background()
	response, err := getPackageByNamingConvention(ctx, client, orgID, name)

	if err != nil {
		return nil, fmt.Errorf("error when querying for package %s - ensure your project, target and package abbreviations are correct:\n\t%w", name, err)
	}

	return response.GetPackageByNamingConvention.toPackage(), nil
}

func (p *getPackageByNamingConventionGetPackageByNamingConventionPackage) toPackage() *Package {
	return &Package{
		NamePrefix: p.NamePrefix,
		Params:     p.Params,
		Manifest: Manifest{
			ID: p.Manifest.Id,
			Bundle: Bundle{
				Name: p.Manifest.Bundle.Name,
			},
		},
		Environment: Environment{
			ID:   p.Environment.Id,
			Slug: p.Environment.Slug,
		},
	}
}

func ConfigurePackage(client graphql.Client, orgID string, targetID string, manifestID string, params map[string]interface{}) (*Package, error) {
	ctx := context.Background()
	response, err := configurePackage(ctx, client, orgID, targetID, manifestID, params)

	if err != nil {
		return nil, err
	}

	if response.ConfigurePackage.Successful {
		return response.ConfigurePackage.Result.toPackage(), nil
	}

	return nil, NewMutationError("failed to configure package", response.ConfigurePackage.Messages)
}

func (p *configurePackageConfigurePackagePackagePayloadResultPackage) toPackage() *Package {
	return &Package{
		ID:         p.Id,
		Params:     p.Params,
		NamePrefix: p.NamePrefix,
	}
}
