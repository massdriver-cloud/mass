package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// ViewerKind discriminates the two flavors of authenticated entity the
// `viewer` query can return.
type ViewerKind string

const (
	// ViewerKindAccount is a human user account (PAT or session auth).
	ViewerKindAccount ViewerKind = "account"
	// ViewerKindServiceAccount is a programmatic service account
	// (basic-auth API key today).
	ViewerKindServiceAccount ViewerKind = "service_account"
)

// Viewer flattens the GraphQL `Viewer` union into a single shape suitable
// for printing. Fields that don't apply to a given kind are left empty
// and omitted from JSON output.
type Viewer struct {
	Kind         ViewerKind          `json:"kind"`
	ID           string              `json:"id"`
	Email        string              `json:"email,omitempty"`
	FirstName    string              `json:"firstName,omitempty"`
	LastName     string              `json:"lastName,omitempty"`
	Name         string              `json:"name,omitempty"`
	Description  string              `json:"description,omitempty"`
	Organization *ViewerOrganization `json:"organization,omitempty"`
}

// ViewerOrganization is a thin reference to the organization tied to the
// viewer — for service accounts this is the owning org; for human users
// this is the most-recently-joined organization.
type ViewerOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetViewer returns the currently authenticated entity. Use it to answer
// "who am I?" — bootstrap UI state, verify which credentials are active,
// or distinguish a user from a service account.
func GetViewer(ctx context.Context, mdClient *client.Client) (*Viewer, error) {
	response, err := getViewer(ctx, mdClient.GQLv2)
	if err != nil {
		return nil, fmt.Errorf("failed to get viewer: %w", err)
	}
	if response.Viewer == nil {
		return nil, errors.New("no authenticated viewer (check MASSDRIVER_API_KEY)")
	}

	switch v := response.Viewer.(type) {
	case *getViewerViewerAccountViewer:
		view := &Viewer{
			Kind:      ViewerKindAccount,
			ID:        v.Id,
			Email:     v.Email,
			FirstName: v.FirstName,
			LastName:  v.LastName,
		}
		if v.DefaultOrganization.Id != "" {
			view.Organization = &ViewerOrganization{ID: v.DefaultOrganization.Id, Name: v.DefaultOrganization.Name}
		}
		return view, nil
	case *getViewerViewerServiceAccountViewer:
		return &Viewer{
			Kind:        ViewerKindServiceAccount,
			ID:          v.Id,
			Name:        v.Name,
			Description: v.Description,
			Organization: &ViewerOrganization{
				ID:   v.Organization.Id,
				Name: v.Organization.Name,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected viewer type: %T", v)
	}
}
