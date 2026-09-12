package config_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestRunRemoveWithForce(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	if err := config.RunRemove(file, "default", true, strings.NewReader(""), &bytes.Buffer{}); err != nil {
		t.Fatalf("RunRemove: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if _, exists := saved.Profiles["default"]; exists {
		t.Error("profile was not removed")
	}
	if _, exists := saved.Profiles["staging"]; !exists {
		t.Error("removing default also removed staging")
	}
}

func TestRunRemoveRequiresTypedConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		removed bool
	}{
		{name: "matching name confirms", input: "default\n", removed: true},
		{name: "wrong name cancels", input: "nope\n", removed: false},
		{name: "empty input cancels", input: "\n", removed: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := loadConfig(t, twoProfiles)

			var out bytes.Buffer
			if err := config.RunRemove(file, "default", false, strings.NewReader(tc.input), &out); err != nil {
				t.Fatalf("RunRemove: %v", err)
			}

			saved, err := sdkconfig.ReadFile()
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			_, exists := saved.Profiles["default"]
			if exists == tc.removed {
				t.Errorf("profile exists = %v, want removed = %v", exists, tc.removed)
			}
			if !tc.removed && !strings.Contains(out.String(), "cancelled") {
				t.Errorf("output = %q, want a cancellation message", out.String())
			}
		})
	}
}

// TestRunRemoveClearsDanglingCurrentProfile matters because the SDK treats
// current_profile as an explicit selection: left pointing at a deleted
// profile, every later command fails with ErrProfileNotFound.
func TestRunRemoveClearsDanglingCurrentProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var out bytes.Buffer
	if err := config.RunRemove(file, "staging", true, strings.NewReader(""), &out); err != nil {
		t.Fatalf("RunRemove: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.CurrentProfile != "" {
		t.Errorf("current_profile = %q, want it cleared", saved.CurrentProfile)
	}
	if !strings.Contains(out.String(), sdkconfig.DefaultProfileName) {
		t.Errorf("output = %q, want it to name the fallback profile", out.String())
	}

	if _, loadErr := sdkconfig.Get(); loadErr != nil {
		t.Errorf("SDK could not resolve after removing the active profile: %v", loadErr)
	}
}

func TestRunRemoveUnknownProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunRemove(file, "nope", true, strings.NewReader(""), &bytes.Buffer{})
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Errorf("RunRemove error = %v, want ErrProfileNotFound", err)
	}
}
