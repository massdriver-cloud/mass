package bundle

import (
	"context"
	"regexp"
	"testing"

	bundlepkg "github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	b := &bundlepkg.Bundle{Name: "test-bundle", Version: "1.0.0"}

	tests := []struct {
		name               string
		existingVersions   []map[string]string
		developmentRelease bool
		wantErr            string
		wantRegexp         string
		wantVersion        string
	}{
		{
			name:               "existing version non-dev",
			existingVersions:   []map[string]string{{"tag": "1.0.0"}},
			developmentRelease: false,
			wantErr:            "version 1.0.0 already exists for bundle test-bundle",
		},
		{
			name:               "existing version development release",
			existingVersions:   []map[string]string{{"tag": "1.0.0"}},
			developmentRelease: true,
			wantErr:            "version 1.0.0 already exists for bundle test-bundle - cannot publish a development release for an existing version",
		},
		{
			name:               "valid version",
			existingVersions:   []map[string]string{},
			developmentRelease: false,
			wantVersion:        "1.0.0",
		},
		{
			name:               "valid development release version",
			existingVersions:   []map[string]string{},
			developmentRelease: true,
			wantRegexp:         `^1\.0\.0-dev\.\d{8}T\d{6}Z$`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare a mock GraphQL client that returns the expected bundle versions
			gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
				"data": map[string]any{
					"ociRepo": map[string]any{
						"tags": tc.existingVersions,
					},
				},
			})
			mdClient := client.Client{
				GQL: gqlClient,
			}

			ver, err := getVersion(context.Background(), &mdClient, b, tc.developmentRelease)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			if tc.wantVersion != "" {
				require.Equal(t, tc.wantVersion, ver)
				return
			}
			if tc.wantRegexp != "" {
				r := regexp.MustCompile(tc.wantRegexp)
				require.True(t, r.MatchString(ver), "version %s did not match regexp %s", ver, tc.wantRegexp)
				return
			}
		})
	}
}
