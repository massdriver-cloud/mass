package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Package struct {
	ID          string                 `json:"id"`
	NamePrefix  string                 `json:"namePrefix"`
	Params      map[string]interface{} `json:"params"`
	Manifest    PackageManifest        `json:"manifest"`
	Environment PackageEnvironment     `json:"environment"`
}

type PackageManifest struct {
	ID     string        `json:"id"`
	Bundle PackageBundle `json:"bundle"`
}

type PackageBundle struct {
	Name string `json:"name"`
}

type PackageEnvironment struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
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
		Manifest: PackageManifest{
			ID: p.Manifest.Id,
			Bundle: PackageBundle{
				Name: p.Manifest.Bundle.Name,
			},
		},
		Environment: PackageEnvironment{
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
