package version_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/version"
)

func TestCheckForNewerVersionAvailable(t *testing.T) {
	tests := []struct {
		name    string
		isOld   bool
		wantErr bool
	}{
		{
			name:    "unknown version should be considered old",
			isOld:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := version.CheckForNewerVersionAvailable()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckForNewerVersionAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.isOld {
				t.Errorf("CheckForNewerVersionAvailable() got = %v, want %v", got, tt.isOld)
			}
		})
	}
}