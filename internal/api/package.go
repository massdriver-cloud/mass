package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Package struct {
	ID         string
	NamePrefix string
	Params     map[string]interface{}
	Manifest   Manifest
	Target     Target
}

func GetPackageByName(client graphql.Client, orgID string, name string) (Package, error) {
	ctx := context.Background()
	response, err := getPackageByNamingConvention(ctx, client, orgID, name)

	if err != nil {
		return Package{}, err
	}

	return response.GetPackageByNamingConvention.toPackage(), nil
}

func (p *getPackageByNamingConventionGetPackageByNamingConventionPackage) toPackage() Package {
	return Package{
		NamePrefix: p.NamePrefix,
		Manifest: Manifest{
			ID: p.Manifest.Id,
		},
		Target: Target{
			ID: p.Target.Id,
		},
	}
}

func ConfigurePackage(client graphql.Client, orgID string, targetID string, manifestID string, params map[string]interface{}) (Package, error) {
	ctx := context.Background()
	pkg := Package{}

	response, err := configurePackage(ctx, client, orgID, targetID, manifestID, params)

	if err != nil {
		return pkg, err
	}

	if response.ConfigurePackage.Successful {
		return response.ConfigurePackage.Result.toPackage(), nil
	}

	msgs, err := json.Marshal(response.ConfigurePackage.Messages)
	if err != nil {
		return pkg, fmt.Errorf("failed to configure package and couldn't marshal error messages: %w", err)
	}

	// TODO: better formatting of errors - custom mutation Error type
	return pkg, fmt.Errorf("failed to configure package: %v", string(msgs))
}

func (p *configurePackageConfigurePackagePackagePayloadResultPackage) toPackage() Package {
	return Package{
		ID:     p.Id,
		Params: p.Params,
	}
}
