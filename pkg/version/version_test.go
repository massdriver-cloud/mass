package version_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/version"
)

func TestCheckForNewerVersionAvailable(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		latest    string
		wantIsOld bool
	}{
		{
			name:      "unknown version should be considered old",
			current:   "unknown",
			latest:    "v1.2.0",
			wantIsOld: true,
		},
		{
			name:      "current version is the latest version",
			current:   "1.2.0",
			latest:    "v1.2.0",
			wantIsOld: false,
		},
		{
			name:      "current version is older than the latest version",
			current:   "1.0.0",
			latest:    "v1.2.0",
			wantIsOld: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version.SetVersion(tt.current)

			got, latestVersion := version.CheckForNewerVersionAvailable(tt.latest)
			if got != tt.wantIsOld {
				t.Errorf("CheckForNewerVersionAvailable() got = %v, want %v", got, tt.wantIsOld)
			}
			if latestVersion != tt.latest {
				t.Errorf("CheckForNewerVersionAvailable() latestVersion = %v, want %v", latestVersion, tt.latest)
			}
		})
	}
}
