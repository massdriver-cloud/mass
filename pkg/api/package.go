package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Package struct {
	ID               string            `json:"id"`
	NamePrefix       string            `json:"namePrefix"`
	Status           string            `json:"status"`
	Artifacts        []Artifact        `json:"artifacts,omitempty"`
	RemoteReferences []RemoteReference `json:"remoteReferences,omitempty"`
	Params           map[string]any    `json:"params"`
	Manifest         *Manifest         `json:"manifest" mapstructure:"manifest,omitempty"`
	Environment      *Environment      `json:"environment,omitempty" mapstructure:"environment,omitempty"`
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

	return toPackage(response.GetPackageByNamingConvention)
}

func toPackage(p any) (*Package, error) {
	pkg := Package{}
	if err := mapstructure.Decode(p, &pkg); err != nil {
		return nil, fmt.Errorf("failed to decode package: %w", err)
	}
	return &pkg, nil
}

func ConfigurePackage(ctx context.Context, mdClient *client.Client, targetID string, manifestID string, params map[string]any) (*Package, error) {
	response, err := configurePackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, targetID, manifestID, params)

	if err != nil {
		return nil, err
	}

	if response.ConfigurePackage.Successful {
		return toPackage(response.ConfigurePackage.Result)
	}

	return nil, NewMutationError("failed to configure package", response.ConfigurePackage.Messages)
}
