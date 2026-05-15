package bundle //nolint:testpackage // needs access to unexported bundle internals

import (
	"regexp"
	"testing"

	bundlepkg "github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	b := &bundlepkg.Bundle{Name: "test-bundle", Version: "1.0.0"}

	tests := []struct {
		name               string
		existingTags       []string
		developmentRelease bool
		wantErr            string
		wantRegexp         string
		wantVersion        string
	}{
		{
			name:               "existing version non-dev",
			existingTags:       []string{"1.0.0"},
			developmentRelease: false,
			wantErr:            "version 1.0.0 already exists for bundle test-bundle",
		},
		{
			name:               "existing version development release",
			existingTags:       []string{"1.0.0"},
			developmentRelease: true,
			wantErr:            "version 1.0.0 already exists for bundle test-bundle - cannot publish a development release for an existing version",
		},
		{
			name:               "valid version",
			existingTags:       []string{},
			developmentRelease: false,
			wantVersion:        "1.0.0",
		},
		{
			name:               "valid development release version",
			existingTags:       []string{},
			developmentRelease: true,
			wantRegexp:         `^1\.0\.0-dev\.\d{8}T\d{6}Z$`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ver, err := resolveVersion(b, tc.existingTags, tc.developmentRelease)
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
