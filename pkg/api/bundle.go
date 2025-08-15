package api

import (
	"context"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type Bundle struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Spec        map[string]any `json:"spec,omitempty"`
	SpecVersion string         `json:"specVersion,omitempty"`
}

func GetBundleVersions(ctx context.Context, mdClient *client.Client, bundleName string) ([]string, error) {
	response, err := getBundleVersions(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleName)
	if err != nil {
		return nil, err
	}
	return response.Bundle.Versions, nil
}
