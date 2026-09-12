package config_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestRunGetDefaultsToActiveProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var out bytes.Buffer
	if err := config.RunGet(file, "", config.GetOptions{Output: "text"}, &out); err != nil {
		t.Fatalf("RunGet: %v", err)
	}
	if !strings.Contains(out.String(), "Profile: staging (active)") {
		t.Errorf("output = %q, want the active profile", out.String())
	}
}

func TestRunGetNamedProfileIsNotMarkedActive(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var out bytes.Buffer
	if err := config.RunGet(file, "default", config.GetOptions{Output: "text"}, &out); err != nil {
		t.Fatalf("RunGet: %v", err)
	}
	if strings.Contains(out.String(), "(active)") {
		t.Errorf("output = %q, want default not marked active", out.String())
	}
}

func TestRunGetMasksAPIKey(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var out bytes.Buffer
	if err := config.RunGet(file, "staging", config.GetOptions{Output: "text"}, &out); err != nil {
		t.Fatalf("RunGet: %v", err)
	}
	if strings.Contains(out.String(), "mds_stagingkey1234") {
		t.Errorf("API key printed in full by default:\n%s", out.String())
	}
}

// TestRunGetUnknownProfileWrapsSDKError keeps the CLI's message classifiable
// the same way the SDK's own profile miss is.
func TestRunGetUnknownProfileWrapsSDKError(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunGet(file, "nope", config.GetOptions{Output: "text"}, &bytes.Buffer{})
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Fatalf("RunGet error = %v, want ErrProfileNotFound", err)
	}
	if !strings.Contains(err.Error(), "default, staging") {
		t.Errorf("error = %v, want it to list the configured profiles", err)
	}
}

func TestRunGetWithNoProfilesSuggestsAdd(t *testing.T) {
	file := loadConfig(t, "")

	err := config.RunGet(file, "staging", config.GetOptions{Output: "text"}, &bytes.Buffer{})
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Fatalf("RunGet error = %v, want ErrProfileNotFound", err)
	}
	if !strings.Contains(err.Error(), "mass config add staging") {
		t.Errorf("error = %v, want it to suggest adding the profile", err)
	}
}
