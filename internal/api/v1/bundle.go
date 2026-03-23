package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Bundle represents a Massdriver bundle (IaC module) and its metadata.
type Bundle struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// GetBundle retrieves a bundle by its identifier (name@version) from the Massdriver API.
func GetBundle(ctx context.Context, mdClient *client.Client, bundleID string) (*Bundle, error) {
	response, err := getBundle(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle %s: %w", bundleID, err)
	}

	b := response.Bundle
	return &Bundle{
		ID:          b.Id,
		Name:        b.Name,
		Version:     b.Version,
		Description: b.Description,
		Icon:        b.Icon,
		SourceURL:   b.SourceUrl,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}, nil
}
