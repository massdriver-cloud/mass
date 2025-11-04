package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Bundle struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Version           string         `json:"version"`
	Description       string         `json:"description,omitempty"`
	Spec              map[string]any `json:"spec,omitempty"`
	SpecVersion       string         `json:"specVersion,omitempty"`
	Icon              string         `json:"icon,omitempty"`
	SourceURL         string         `json:"sourceUrl,omitempty"`
	ParamsSchema      map[string]any `json:"paramsSchema,omitempty"`
	ConnectionsSchema map[string]any `json:"connectionsSchema,omitempty"`
	ArtifactsSchema   map[string]any `json:"artifactsSchema,omitempty"`
	UISchema          map[string]any `json:"uiSchema,omitempty"`
	OperatorGuide     string         `json:"operatorGuide,omitempty"`
	CreatedAt         time.Time      `json:"createdAt,omitempty"`
	UpdatedAt         time.Time      `json:"updatedAt,omitempty"`
}

func GetBundle(ctx context.Context, mdClient *client.Client, bundleId string, version *string) (*Bundle, error) {
	versionStr := ""
	if version != nil {
		versionStr = *version
	}
	response, err := getBundle(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleId, versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle %s: %w", bundleId, err)
	}
	return toBundle(response.Bundle)
}

func toBundle(b any) (*Bundle, error) {
	// Type assert to the generated type
	genBundle, ok := b.(getBundleBundle)
	if !ok {
		// Fallback to mapstructure for flexibility
		bundle := Bundle{}
		if err := mapstructure.Decode(b, &bundle); err != nil {
			return nil, fmt.Errorf("failed to decode bundle: %w", err)
		}
		return &bundle, nil
	}

	// Direct assignment from generated type
	bundle := Bundle{
		ID:                genBundle.Id,
		Name:              genBundle.Name,
		Version:           genBundle.Version,
		Description:       genBundle.Description,
		Spec:              genBundle.Spec,
		SpecVersion:       genBundle.SpecVersion,
		Icon:              genBundle.Icon,
		SourceURL:         genBundle.SourceUrl,
		ParamsSchema:      genBundle.ParamsSchema,
		ConnectionsSchema: genBundle.ConnectionsSchema,
		ArtifactsSchema:   genBundle.ArtifactsSchema,
		UISchema:          genBundle.UiSchema,
		OperatorGuide:     genBundle.OperatorGuide,
		CreatedAt:         genBundle.CreatedAt,
		UpdatedAt:         genBundle.UpdatedAt,
	}
	return &bundle, nil
}
