package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Server holds information about a Massdriver server.
type Server struct {
	AppURL           string            `json:"appUrl" mapstructure:"appUrl"`
	Version          string            `json:"version" mapstructure:"version"`
	Mode             string            `json:"mode" mapstructure:"mode"`
	SsoProviders     []SsoProvider     `json:"ssoProviders,omitempty" mapstructure:"ssoProviders"`
	EmailAuthMethods []EmailAuthMethod `json:"emailAuthMethods,omitempty" mapstructure:"emailAuthMethods"`
}

// SsoProvider is an SSO provider available for authentication.
type SsoProvider struct {
	Name      string `json:"name" mapstructure:"name"`
	LoginURL  string `json:"loginUrl" mapstructure:"loginUrl"`
	UIIconURL string `json:"uiIconUrl,omitempty" mapstructure:"uiIconUrl"`
	UILabel   string `json:"uiLabel,omitempty" mapstructure:"uiLabel"`
}

// EmailAuthMethod is an email-based authentication method.
type EmailAuthMethod struct {
	Name string `json:"name" mapstructure:"name"`
}

// GetServer returns server info and available authentication methods. No authentication required.
func GetServer(ctx context.Context, mdClient *client.Client) (*Server, error) {
	response, err := getServer(ctx, mdClient.GQLv1)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	return toServer(response.Server)
}

func toServer(v any) (*Server, error) {
	s := Server{}
	if err := mapstructure.Decode(v, &s); err != nil {
		return nil, fmt.Errorf("failed to decode server: %w", err)
	}
	return &s, nil
}
