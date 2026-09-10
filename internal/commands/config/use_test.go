package config_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// TestRunUseIsHonoredBySDK is the contract this command rests on: it writes
// current_profile, and the SDK — with no override and no environment variable
// — selects it.
func TestRunUseIsHonoredBySDK(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	if err := config.RunUse(file, "default", &bytes.Buffer{}); err != nil {
		t.Fatalf("RunUse: %v", err)
	}

	cfg, err := sdkconfig.Get()
	if err != nil {
		t.Fatalf("sdkconfig.Get: %v", err)
	}
	if cfg.Profile != "default" {
		t.Errorf("SDK selected %q, want default", cfg.Profile)
	}
}

// TestRunUseRejectsUnknownProfile keeps a typo from writing a current_profile
// that would fail every later command.
func TestRunUseRejectsUnknownProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunUse(file, "nope", &bytes.Buffer{})
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Fatalf("RunUse error = %v, want ErrProfileNotFound", err)
	}

	saved, readErr := sdkconfig.ReadFile()
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if saved.CurrentProfile != "staging" {
		t.Errorf("current_profile = %q, want staging to be untouched", saved.CurrentProfile)
	}
}

// TestRunUseWarnsWhenEnvironmentOverrides tells the user why their switch
// appears to have had no effect.
func TestRunUseWarnsWhenEnvironmentOverrides(t *testing.T) {
	file := loadConfig(t, twoProfiles)
	t.Setenv("MASSDRIVER_PROFILE", "staging")

	var out bytes.Buffer
	if err := config.RunUse(file, "default", &out); err != nil {
		t.Fatalf("RunUse: %v", err)
	}
	if !strings.Contains(out.String(), "MASSDRIVER_PROFILE") {
		t.Errorf("output = %q, want a warning about the environment variable", out.String())
	}
}
