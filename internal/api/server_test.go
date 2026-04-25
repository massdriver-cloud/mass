package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetServer(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"server": map[string]any{
				"appUrl":  "https://app.massdriver.cloud",
				"version": "1.2.3",
				"mode":    "MANAGED",
				"ssoProviders": []map[string]any{
					{
						"name":      "google",
						"loginUrl":  "https://app.massdriver.cloud/auth/google",
						"uiIconUrl": "https://app.massdriver.cloud/icons/google.svg",
						"uiLabel":   "Sign in with Google",
					},
				},
				"emailAuthMethods": []map[string]any{
					{"name": "PASSKEY"},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	server, err := api.GetServer(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	if server.AppURL != "https://app.massdriver.cloud" {
		t.Errorf("got %s, wanted https://app.massdriver.cloud", server.AppURL)
	}
	if server.Version != "1.2.3" {
		t.Errorf("got %s, wanted 1.2.3", server.Version)
	}
	if server.Mode != "MANAGED" {
		t.Errorf("got %s, wanted MANAGED", server.Mode)
	}
	if len(server.SsoProviders) != 1 || server.SsoProviders[0].Name != "google" {
		t.Errorf("expected one google SSO provider, got %+v", server.SsoProviders)
	}
	if server.SsoProviders[0].LoginURL != "https://app.massdriver.cloud/auth/google" {
		t.Errorf("got login URL %s", server.SsoProviders[0].LoginURL)
	}
	if len(server.EmailAuthMethods) != 1 || server.EmailAuthMethods[0].Name != "PASSKEY" {
		t.Errorf("expected one PASSKEY email auth method, got %+v", server.EmailAuthMethods)
	}
}
