package api

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID                string         `json:"id" mapstructure:"id"`
	Name              string         `json:"name" mapstructure:"name"`
	Version           string         `json:"version" mapstructure:"version"`
	Description       string         `json:"description,omitempty" mapstructure:"description"`
	Spec              map[string]any `json:"spec,omitempty" mapstructure:"spec"`
	SpecVersion       string         `json:"specVersion,omitempty" mapstructure:"specVersion"`
	Icon              string         `json:"icon,omitempty" mapstructure:"icon"`
	SourceURL         string         `json:"sourceUrl,omitempty" mapstructure:"sourceUrl"`
	ParamsSchema      map[string]any `json:"paramsSchema,omitempty" mapstructure:"paramsSchema"`
	ConnectionsSchema map[string]any `json:"connectionsSchema,omitempty" mapstructure:"connectionsSchema"`
	ArtifactsSchema   map[string]any `json:"artifactsSchema,omitempty" mapstructure:"artifactsSchema"`
	UISchema          map[string]any `json:"uiSchema,omitempty" mapstructure:"uiSchema"`
	OperatorGuide     string         `json:"operatorGuide,omitempty" mapstructure:"operatorGuide"`
	CreatedAt         time.Time      `json:"createdAt,omitempty" mapstructure:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt,omitempty" mapstructure:"updatedAt"`
}

// // GetBundle retrieves a bundle by ID and optional version from the Massdriver API.
// func GetBundle(ctx context.Context, mdClient *client.Client, bundleID string, version *string) (*Bundle, error) {
// 	versionStr := ""
// 	if version != nil {
// 		versionStr = *version
// 	}
// 	response, err := getBundle(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleID, versionStr)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get bundle %s: %w", bundleID, err)
// 	}
// 	return toBundle(response.Bundle)
// }

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
