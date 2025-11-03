package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Bundle struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Spec        map[string]any `json:"spec,omitempty"`
	SpecVersion string         `json:"specVersion,omitempty"`
}

func GetBundle(ctx context.Context, mdClient *client.Client, bundleId string) (*Bundle, error) {
	response, err := getBundle(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleId)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle %s: %w", bundleId, err)
	}
	return toBundle(response.Bundle)
}

func toBundle(b any) (*Bundle, error) {
	bundle := Bundle{}
	if err := mapstructure.Decode(b, &bundle); err != nil {
		return nil, fmt.Errorf("failed to decode bundle: %w", err)
	}
	return &bundle, nil
}
