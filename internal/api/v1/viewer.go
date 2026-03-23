package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// ViewerOrganization represents an organization the viewer belongs to.
type ViewerOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AccountViewer represents an authenticated human user.
type AccountViewer struct {
	ID                  string              `json:"id"`
	Email               string              `json:"email"`
	FirstName           string              `json:"firstName"`
	LastName            string              `json:"lastName"`
	CreatedAt           time.Time           `json:"createdAt"`
	UpdatedAt           time.Time           `json:"updatedAt"`
	DefaultOrganization *ViewerOrganization `json:"defaultOrganization,omitempty"`
}

// ServiceAccountViewer represents an authenticated API client.
type ServiceAccountViewer struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
	Organization *ViewerOrganization `json:"organization"`
}

// Viewer is the result of the getViewer query. Exactly one field will be non-nil.
type Viewer struct {
	Account        *AccountViewer
	ServiceAccount *ServiceAccountViewer
}

// GetViewer retrieves information about the authenticated user or service account.
func GetViewer(ctx context.Context, mdClient *client.Client) (*Viewer, error) {
	response, err := getViewer(ctx, mdClient.GQL)
	if err != nil {
		return nil, fmt.Errorf("failed to get viewer: %w", err)
	}

	viewer := &Viewer{}
	switch v := response.Viewer.(type) {
	case *getViewerViewerAccountViewer:
		viewer.Account = &AccountViewer{
			ID:        v.Id,
			Email:     v.Email,
			FirstName: v.FirstName,
			LastName:  v.LastName,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
			DefaultOrganization: &ViewerOrganization{
				ID:   v.DefaultOrganization.Id,
				Name: v.DefaultOrganization.Name,
			},
		}
	case *getViewerViewerServiceAccountViewer:
		viewer.ServiceAccount = &ServiceAccountViewer{
			ID:          v.Id,
			Name:        v.Name,
			Description: v.Description,
			CreatedAt:   v.CreatedAt,
			UpdatedAt:   v.UpdatedAt,
			Organization: &ViewerOrganization{
				ID:   v.Organization.Id,
				Name: v.Organization.Name,
			},
		}
	default:
		return nil, fmt.Errorf("unexpected viewer type: %T", v)
	}

	return viewer, nil
}
