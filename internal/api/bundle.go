package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID           string       `json:"id" mapstructure:"id"`
	Name         string       `json:"name" mapstructure:"name"`
	Version      string       `json:"version" mapstructure:"version"`
	Description  string       `json:"description,omitempty" mapstructure:"description"`
	Icon         string       `json:"icon,omitempty" mapstructure:"icon"`
	SourceURL    string       `json:"sourceUrl,omitempty" mapstructure:"sourceUrl"`
	Repo         string       `json:"repo,omitempty" mapstructure:"repo"`
	CreatedAt    time.Time    `json:"createdAt,omitzero" mapstructure:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt,omitzero" mapstructure:"updatedAt"`
	Dependencies []BundleSlot `json:"dependencies,omitempty" mapstructure:"dependencies"`
	Resources    []BundleSlot `json:"resources,omitempty" mapstructure:"resources"`
}

// BundleSlot describes one of a bundle's input dependencies or output
// resources. Dependencies are slots the user must wire up; resources are
// outputs the bundle produces on a successful deployment.
type BundleSlot struct {
	Name         string             `json:"name" mapstructure:"name"`
	Required     bool               `json:"required" mapstructure:"required"`
	ResourceType *BundleResourceRef `json:"resourceType,omitempty" mapstructure:"resourceType"`
}

// BundleResourceRef points at a resource type the bundle declares. May be
// nil when the original resource type has been removed from the catalog.
type BundleResourceRef struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

// GetBundle retrieves a bundle by its identifier (e.g., "aws-aurora-postgres@1.2.3" or "aws-aurora-postgres@latest").
func GetBundle(ctx context.Context, mdClient *client.Client, id string) (*Bundle, error) {
	response, err := getBundle(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle %s: %w", id, err)
	}
	return toBundle(response.Bundle)
}

// ListBundles returns bundles, optionally filtered and sorted.
func ListBundles(ctx context.Context, mdClient *client.Client, filter *BundlesFilter, sort *BundlesSort) ([]Bundle, error) {
	response, err := listBundles(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, filter, sort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list bundles: %w", err)
	}

	bundles := make([]Bundle, 0, len(response.Bundles.Items))
	for _, resp := range response.Bundles.Items {
		b, bErr := toBundle(resp)
		if bErr != nil {
			return nil, fmt.Errorf("failed to convert bundle: %w", bErr)
		}
		bundles = append(bundles, *b)
	}

	return bundles, nil
}

func toBundle(v any) (*Bundle, error) {
	b := Bundle{}
	if err := decode(v, &b); err != nil {
		return nil, fmt.Errorf("failed to decode bundle: %w", err)
	}
	return &b, nil
}
