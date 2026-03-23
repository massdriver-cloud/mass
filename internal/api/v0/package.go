package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// PackageDeployment represents a deployment summary embedded in a package response.
type PackageDeployment struct {
	ID        string    `json:"id" mapstructure:"id"`
	Status    string    `json:"status" mapstructure:"status"`
	Action    string    `json:"action" mapstructure:"action"`
	Version   string    `json:"version" mapstructure:"version"`
	CreatedAt time.Time `json:"createdAt" mapstructure:"createdAt"`
}

// Package represents a deployed bundle instance within a Massdriver environment.
type Package struct {
	ID               string             `json:"id" mapstructure:"id"`
	Slug             string             `json:"slug" mapstructure:"slug"`
	Status           string             `json:"status" mapstructure:"status"`
	DeployedVersion  *string            `json:"deployedVersion,omitempty" mapstructure:"deployedVersion"`
	LatestDeployment *PackageDeployment `json:"latestDeployment,omitempty" mapstructure:"latestDeployment"`
	ActiveDeployment *PackageDeployment `json:"activeDeployment,omitempty" mapstructure:"activeDeployment"`
	Artifacts        []Artifact         `json:"artifacts,omitempty" mapstructure:"artifacts"`
	RemoteReferences []RemoteReference  `json:"remoteReferences,omitempty" mapstructure:"remoteReferences"`
	Bundle           *Bundle            `json:"bundle,omitempty" mapstructure:"bundle,omitempty"`
	Params           map[string]any     `json:"params" mapstructure:"params"`
	Manifest         *Manifest          `json:"manifest" mapstructure:"manifest,omitempty"`
	Environment      *Environment       `json:"environment,omitempty" mapstructure:"environment,omitempty"`
}

// ParamsJSON returns the package parameters serialized as a pretty-printed JSON string.
func (p *Package) ParamsJSON() (string, error) {
	paramsJSON, err := json.MarshalIndent(p.Params, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal params to JSON: %w", err)
	}
	return string(paramsJSON), nil
}

// GetPackage retrieves a package by slug or ID from the Massdriver API.
func GetPackage(ctx context.Context, mdClient *client.Client, name string) (*Package, error) {
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

	// mapstructure creates empty structs/pointers for nil values; nil them out
	if pkg.DeployedVersion != nil && *pkg.DeployedVersion == "" {
		pkg.DeployedVersion = nil
	}
	if pkg.LatestDeployment != nil && pkg.LatestDeployment.ID == "" {
		pkg.LatestDeployment = nil
	}
	if pkg.ActiveDeployment != nil && pkg.ActiveDeployment.ID == "" {
		pkg.ActiveDeployment = nil
	}

	return &pkg, nil
}

// ConfigurePackage updates the configuration parameters of a package in the Massdriver API.
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

// SetPackageVersion sets the bundle version and release strategy for a package.
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

// DecommissionPackage initiates decommissioning of a package and all its resources.
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

// ResetPackage resets the deployment state of a package in the Massdriver API.
func ResetPackage(ctx context.Context, mdClient *client.Client, id string) (*Package, error) {
	deleteState := false
	deleteParams := false
	deleteDeployments := true

	response, err := resetPackage(ctx, mdClient.GQL, mdClient.Config.OrganizationID, id, deleteState, deleteParams, deleteDeployments)

	if err != nil {
		return nil, err
	}

	if response.ResetPackage.Successful {
		return toPackage(response.ResetPackage.Result)
	}

	return nil, NewMutationError("failed to reset package", response.ResetPackage.Messages)
}
