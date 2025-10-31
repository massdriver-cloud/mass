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
		name             string
		existingVersions []string
		releaseCandidate bool
		wantErr          string
		wantRegexp       string
		wantVersion      string
	}{
		{
			name:             "existing version non-rc",
			existingVersions: []string{"1.0.0"},
			releaseCandidate: false,
			wantErr:          "version 1.0.0 already exists for bundle test-bundle",
		},
		{
			name:             "existing version rc",
			existingVersions: []string{"1.0.0"},
			releaseCandidate: true,
			wantErr:          "version 1.0.0 already exists for bundle test-bundle - cannot publish a release candidate for an existing version",
		},
		{
			name:             "valid version",
			existingVersions: []string{},
			releaseCandidate: false,
			wantVersion:      "1.0.0",
		},
		{
			name:             "valid rc",
			existingVersions: []string{},
			releaseCandidate: true,
			wantRegexp:       `^1\.0\.0-rc\.\d{8}T\d{6}Z$`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare a mock GraphQL client that returns the expected bundle versions
			gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
				"data": map[string]any{
					"bundle": map[string]any{
						"versions": tc.existingVersions,
					},
				},
			})
			mdClient := client.Client{
				GQL: gqlClient,
			}

			ver, err := getVersion(context.Background(), &mdClient, b, tc.releaseCandidate)
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
