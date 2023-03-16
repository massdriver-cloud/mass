package api

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type Package struct {
	ID         string
	NamePrefix string
	Params     map[string]interface{}
	Manifest   Manifest
	Target     Target
}

func GetPackageByName(client graphql.Client, orgID string, name string) (*Package, error) {
	ctx := context.Background()
	response, err := getPackageByNamingConvention(ctx, client, orgID, name)

	if err != nil {
		return nil, err
	}

	return response.GetPackageByNamingConvention.toPackage(), nil
}

func (p *getPackageByNamingConventionGetPackageByNamingConventionPackage) toPackage() *Package {
	return &Package{
		NamePrefix: p.NamePrefix,
		Manifest: Manifest{
			ID: p.Manifest.Id,
		},
		Target: Target{
			ID: p.Target.Id,
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
