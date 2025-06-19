package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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

func GetPackageByName(ctx context.Context, mdClient *client.Client, name string) (*Package, error) {
	response, err := getPackageByNamingConvention(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name)

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

func ConfigurePackage(ctx context.Context, mdClient *client.Client, targetID string, manifestID string, params map[string]interface{}) (*Package, error) {
	response, err := configurePackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, targetID, manifestID, params)

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
