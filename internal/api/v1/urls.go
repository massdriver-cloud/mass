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
	appURL := strings.Replace(mdClient.Config.URL, "api.", "app.", 1)
	server, err := GetServer(ctx, mdClient)
	if err == nil {
		appURL = server.AppURL
	}
	return &URLHelper{
		baseURL: appURL,
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
func (u *URLHelper) ProjectURL(projectID string) string {
	return fmt.Sprintf("%s/orgs/%s/projects/%s/", u.baseURL, u.orgID, projectID)
}

// EnvironmentURL returns the URL for a specific environment
func (u *URLHelper) EnvironmentURL(environmentID string) string {
	parts := strings.Split(environmentID, "-")
	return fmt.Sprintf("%s/orgs/%s/projects/%s/environments/%s", u.baseURL, u.orgID, parts[0], parts[1])
}

// InstanceURL returns the URL for a specific package
func (u *URLHelper) InstanceURL(instanceID string) string {
	parts := strings.Split(instanceID, "-")
	return fmt.Sprintf("%s/orgs/%s/projects/%s/environments/%s?package=%s", u.baseURL, u.orgID, parts[0], parts[1], parts[2])
}

// BundleURL returns the URL for a specific bundle version
func (u *URLHelper) BundleURL(bundleName, version string) string {
	return fmt.Sprintf("%s/orgs/%s/repos/%s/%s", u.baseURL, u.orgID, bundleName, version)
}

// RepoInstancesURL returns the URL for bundle instances
func (u *URLHelper) RepoInstancesURL(bundleName, version string) string {
	return fmt.Sprintf("%s/orgs/%s/repos/%s/%s/instances", u.baseURL, u.orgID, bundleName, version)
}
