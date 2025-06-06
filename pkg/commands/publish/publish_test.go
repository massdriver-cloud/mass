package publish

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

func TestGetRepo(t *testing.T) {
	tests := []struct {
		name     string
		bundle   *bundle.Bundle
		mdClient *client.Client
		wantRepo *remote.Repository
		wantErr  bool
	}{
		{
			name: "valid bundle and client",
			bundle: &bundle.Bundle{
				Name: "test-bundle",
			},
			mdClient: &client.Client{
				Auth: &config.Auth{
					Method:    config.AuthAPIKey,
					URL:       "api.massdriver.cloud",
					AccountID: "sandbox",
					Value:     "p@ssw0rd",
				},
			},
			wantRepo: &remote.Repository{
				Reference: registry.Reference{
					Registry:   "api.massdriver.cloud",
					Repository: "sandbox/test-bundle",
				},
			},
			wantErr: false,
		},
		{
			name: "wrong auth",
			bundle: &bundle.Bundle{
				Name: "test-bundle",
			},
			mdClient: &client.Client{
				Auth: &config.Auth{
					Method:    config.AuthDeployment,
					URL:       "api.massdriver.cloud",
					AccountID: "sandbox",
					Value:     "p@ssw0rd",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRepo(tt.bundle, tt.mdClient)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			repo, ok := got.(*remote.Repository)
			if !ok {
				t.Fatalf("expected *remote.Repository, got %T", got)
			}
			if repo.Reference != tt.wantRepo.Reference {
				t.Errorf("Registry = %v, want %v", repo.Reference, tt.wantRepo.Reference)
			}
		})
	}
}
