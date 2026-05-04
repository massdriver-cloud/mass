package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID          string    `json:"id" mapstructure:"id"`
	Name        string    `json:"name" mapstructure:"name"`
	Version     string    `json:"version" mapstructure:"version"`
	Description string    `json:"description,omitempty" mapstructure:"description"`
	Icon        string    `json:"icon,omitempty" mapstructure:"icon"`
	SourceURL   string    `json:"sourceUrl,omitempty" mapstructure:"sourceUrl"`
	Repo        string    `json:"repo,omitempty" mapstructure:"repo"`
	CreatedAt   time.Time `json:"createdAt,omitzero" mapstructure:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitzero" mapstructure:"updatedAt"`
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
