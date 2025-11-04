package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// URLHelper provides methods to generate Massdriver app URLs
type URLHelper struct {
	baseURL string
	orgID   string
}

// NewURLHelper creates a new URLHelper instance
func NewURLHelper(ctx context.Context, mdClient *client.Client) (*URLHelper, error) {
	server, err := GetServer(ctx, mdClient)
	if err != nil {
		// Fallback: try to derive from API URL
		appURL := strings.Replace(mdClient.Config.URL, "api.", "app.", 1)
		return &URLHelper{
			baseURL: appURL,
			orgID:   mdClient.Config.OrganizationID,
		}, nil
	}

	return &URLHelper{
		baseURL: server.AppURL,
		orgID:   mdClient.Config.OrganizationID,
	}, nil
}

// OrganizationURL returns the URL for an organization
func (u *URLHelper) OrganizationURL() string {
	return fmt.Sprintf("%s/orgs/%s/", u.baseURL, u.orgID)
}

// ProjectsURL returns the URL for listing projects
func (u *URLHelper) ProjectsURL() string {
	return fmt.Sprintf("%s/orgs/%s/projects", u.baseURL, u.orgID)
}

// ProjectURL returns the URL for a specific project
func (u *URLHelper) ProjectURL(projectSlug string) string {
	return fmt.Sprintf("%s/orgs/%s/projects/%s/", u.baseURL, u.orgID, projectSlug)
}

// EnvironmentURL returns the URL for a specific environment
func (u *URLHelper) EnvironmentURL(projectSlug, environmentSlug string) string {
	return fmt.Sprintf("%s/orgs/%s/projects/%s/environments/%s", u.baseURL, u.orgID, projectSlug, environmentSlug)
}

// PackageURL returns the URL for a specific package
func (u *URLHelper) PackageURL(projectSlug, environmentSlug, packageSlug string) string {
	return fmt.Sprintf("%s/orgs/%s/projects/%s/environments/%s?package=%s", u.baseURL, u.orgID, projectSlug, environmentSlug, packageSlug)
}
