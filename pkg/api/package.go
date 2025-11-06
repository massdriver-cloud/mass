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
	Slug             string            `json:"slug"`
	Status           string            `json:"status"`
	Artifacts        []Artifact        `json:"artifacts,omitempty"`
	RemoteReferences []RemoteReference `json:"remoteReferences,omitempty"`
	Bundle           *Bundle           `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
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
	response, err := getPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name)
	if err != nil {
		return nil, fmt.Errorf("error when querying for package %s - ensure your project, target and package abbreviations are correct:\n\t%w", name, err)
	}

	return toPackage(response.Package)
}

func toPackage(p any) (*Package, error) {
	pkg := Package{}
	if err := mapstructure.Decode(p, &pkg); err != nil {
		return nil, fmt.Errorf("failed to decode package: %w", err)
	}
	return &pkg, nil
}

func ConfigurePackage(ctx context.Context, mdClient *client.Client, name string, params map[string]any) (*Package, error) {
	response, err := configurePackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, name, params)

	if err != nil {
		return nil, err
	}

	if response.ConfigurePackage.Successful {
		return toPackage(response.ConfigurePackage.Result)
	}

	return nil, NewMutationError("failed to configure package", response.ConfigurePackage.Messages)
}

func SetPackageVersion(ctx context.Context, mdClient *client.Client, id string, version string, releaseStrategy ReleaseStrategy) (*Package, error) {
	response, err := setPackageVersion(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id, version, releaseStrategy)

	if err != nil {
		return nil, err
	}

	if response.SetPackageVersion.Successful {
		return toPackage(response.SetPackageVersion.Result)
	}

	return nil, NewMutationError("failed to set package version", response.SetPackageVersion.Messages)
}

func DecommissionPackage(ctx context.Context, mdClient *client.Client, id string, message string) (*Deployment, error) {
	response, err := decommissionPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id, message)

	if err != nil {
		return nil, err
	}

	if response.DecommissionPackage.Successful {
		return response.DecommissionPackage.Result.toDeployment(), nil
	}

	return nil, NewMutationError("failed to decommission package", response.DecommissionPackage.Messages)
}
