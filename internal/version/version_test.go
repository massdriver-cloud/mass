package version_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/version"
)

func TestCheckForNewerVersionAvailable(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		latest    string
		wantIsOld bool
		wantErr   bool
	}{
		{
			name:      "unknown version should be considered old",
			current:   "unknown",
			latest:    "v1.2.0",
			wantIsOld: true,
			wantErr:   false,
		},
		{
			name:      "current version is the latest version",
			current:   "1.2.0",
			latest:    "v1.2.0",
			wantIsOld: false,
			wantErr:   false,
		},
		{
			name:      "current version is older than the latest version",
			current:   "1.0.0",
			latest:    "v1.2.0",
			wantIsOld: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version.SetVersion(tt.current)
			version.GetLatestVersion = func() (string, error) {
				return tt.latest, nil
			}
			got, latestVersion, err := version.CheckForNewerVersionAvailable()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckForNewerVersionAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantIsOld {
				t.Errorf("CheckForNewerVersionAvailable() got = %v, want %v", got, tt.wantIsOld)
			}
			if latestVersion != tt.latest {
				t.Errorf("CheckForNewerVersionAvailable() latestVersion = %v, want %v", latestVersion, tt.latest)
			}
		})
	}
}
