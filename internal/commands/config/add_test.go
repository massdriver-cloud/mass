package config_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestRunAddWritesProfile(t *testing.T) {
	file := loadConfig(t, "")

	opts := config.AddOptions{Edits: config.ProfileEdits{
		OrganizationID: ptr("acme"),
		APIKey:         ptr("mds_key123456789"),
		URL:            ptr("https://api.massdriver.example.com"),
	}}
	if err := config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("SDK could not read the file we wrote: %v", err)
	}
	want := sdkconfig.Profile{
		OrganizationID: "acme",
		APIKey:         "mds_key123456789",
		URL:            "https://api.massdriver.example.com",
	}
	if got := saved.Profiles["staging"]; got != want {
		t.Errorf("saved profile = %+v, want %+v", got, want)
	}
}

// TestRunAddAdoptsFirstProfile covers the case where not setting
// current_profile would leave every command looking for a "default" that
// doesn't exist.
func TestRunAddAdoptsFirstProfile(t *testing.T) {
	file := loadConfig(t, "")

	opts := config.AddOptions{Edits: config.ProfileEdits{
		OrganizationID: ptr("acme"),
		APIKey:         ptr("mds_key123456789"),
	}}
	var out bytes.Buffer
	if err := config.RunAdd(t.Context(), file, "staging", opts, &out); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.CurrentProfile != "staging" {
		t.Errorf("current_profile = %q, want staging", saved.CurrentProfile)
	}
	if !strings.Contains(out.String(), "Active profile is now `staging`") {
		t.Errorf("output = %q, want it to report the active profile", out.String())
	}
}

func TestRunAddLeavesActiveProfileAloneByDefault(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	opts := config.AddOptions{Edits: config.ProfileEdits{
		OrganizationID: ptr("acme"),
		APIKey:         ptr("mds_key123456789"),
	}}
	if err := config.RunAdd(t.Context(), file, "production", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.CurrentProfile != "staging" {
		t.Errorf("current_profile = %q, want staging to be untouched", saved.CurrentProfile)
	}
}

func TestRunAddUseFlagSwitchesActiveProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	opts := config.AddOptions{
		Edits: config.ProfileEdits{OrganizationID: ptr("acme"), APIKey: ptr("mds_key123456789")},
		Use:   true,
	}
	if err := config.RunAdd(t.Context(), file, "production", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.CurrentProfile != "production" {
		t.Errorf("current_profile = %q, want production", saved.CurrentProfile)
	}
}

func TestRunAddRefusesToOverwriteWithoutForce(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	opts := config.AddOptions{Edits: config.ProfileEdits{
		OrganizationID: ptr("other"),
		APIKey:         ptr("mds_other12345678"),
	}}
	err := config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunAdd overwrote an existing profile without --force")
	}
	if !strings.Contains(err.Error(), "mass config set staging") {
		t.Errorf("error = %v, want it to point at `mass config set`", err)
	}

	opts.Force = true
	if err = config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd with --force: %v", err)
	}
	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.Profiles["staging"].OrganizationID != "other" {
		t.Error("--force did not overwrite the profile")
	}
}

func TestRunAddRequiresCredentials(t *testing.T) {
	tests := []struct {
		name  string
		edits config.ProfileEdits
		want  string
	}{
		{name: "missing organization", edits: config.ProfileEdits{APIKey: ptr("mds_key123456789")}, want: "--org"},
		{name: "missing API key", edits: config.ProfileEdits{OrganizationID: ptr("acme")}, want: "--api-key"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := loadConfig(t, "")

			err := config.RunAdd(t.Context(), file, "staging", config.AddOptions{Edits: tc.edits}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("RunAdd succeeded with incomplete credentials")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to name %s", err, tc.want)
			}
		})
	}
}

func TestRunAddPromptsForMissingValues(t *testing.T) {
	file := loadConfig(t, "")

	prompter := &stubPrompter{answers: []string{"acme", "mds_key123456789"}}
	opts := config.AddOptions{Prompter: prompter}
	if err := config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	if len(prompter.labels) != 2 {
		t.Fatalf("prompted %d times, want 2: %v", len(prompter.labels), prompter.labels)
	}
	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.Profiles["staging"].OrganizationID != "acme" {
		t.Errorf("prompted organization was not saved: %+v", saved.Profiles["staging"])
	}
}

func TestRunAddDoesNotPromptForValuesAlreadyPassed(t *testing.T) {
	file := loadConfig(t, "")

	prompter := &stubPrompter{}
	opts := config.AddOptions{
		Edits:    config.ProfileEdits{OrganizationID: ptr("acme"), APIKey: ptr("mds_key123456789")},
		Prompter: prompter,
	}
	if err := config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	if len(prompter.labels) != 0 {
		t.Errorf("prompted for values that were passed as flags: %v", prompter.labels)
	}
}

func TestRunAddVerifiesBeforeWriting(t *testing.T) {
	file := loadConfig(t, "")

	verifier := &stubVerifier{identity: "someone@example.com"}
	opts := config.AddOptions{
		Edits:    config.ProfileEdits{OrganizationID: ptr("acme"), APIKey: ptr("mds_key123456789")},
		Verifier: verifier,
	}
	var out bytes.Buffer
	if err := config.RunAdd(t.Context(), file, "staging", opts, &out); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	if verifier.calls != 1 {
		t.Errorf("verifier called %d times, want 1", verifier.calls)
	}
	if verifier.got.APIKey != "mds_key123456789" {
		t.Errorf("verified %+v, want the profile being added", verifier.got)
	}
	if !strings.Contains(out.String(), "someone@example.com") {
		t.Errorf("output = %q, want the verified identity", out.String())
	}
}

// TestRunAddDoesNotWriteWhenVerificationFails is the guarantee that a mistyped
// key never lands in the config file.
func TestRunAddDoesNotWriteWhenVerificationFails(t *testing.T) {
	file := loadConfig(t, "")

	verifier := &stubVerifier{err: errors.New("unauthorized")}
	opts := config.AddOptions{
		Edits:    config.ProfileEdits{OrganizationID: ptr("acme"), APIKey: ptr("mds_key123456789")},
		Verifier: verifier,
	}
	err := config.RunAdd(t.Context(), file, "staging", opts, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunAdd succeeded despite failed verification")
	}
	if !strings.Contains(err.Error(), "--no-verify") {
		t.Errorf("error = %v, want it to mention --no-verify", err)
	}

	path, err := configfile.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("config file was created despite failed verification")
	}
}
