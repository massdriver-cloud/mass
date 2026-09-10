package config_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestRunSetChangesOnlyWhatWasPassed(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	edits := config.ProfileEdits{APIKey: ptr("mds_rotated123456")}
	if err := config.RunSet(t.Context(), file, "staging", edits, nil, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSet: %v", err)
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := sdkconfig.Profile{
		OrganizationID: "acme",
		APIKey:         "mds_rotated123456",
		URL:            "https://api.massdriver.example.com",
	}
	if got := saved.Profiles["staging"]; got != want {
		t.Errorf("saved profile = %+v, want %+v", got, want)
	}
}

func TestRunSetClearsFieldOnExplicitEmptyValue(t *testing.T) {
	file := loadConfig(t, `version: 1
profiles:
  staging:
    organization_id: acme
    api_key: mds_stagingkey1234
    templates_path: /tmp/templates
`)

	edits := config.ProfileEdits{TemplatesPath: ptr("")}
	if err := config.RunSet(t.Context(), file, "staging", edits, nil, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSet: %v", err)
	}

	path, err := configfile.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "templates_path") {
		t.Errorf("cleared field was written to the file:\n%s", data)
	}
}

func TestRunSetRequiresAtLeastOneEdit(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunSet(t.Context(), file, "staging", config.ProfileEdits{}, nil, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunSet with no edits succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "--org") {
		t.Errorf("error = %v, want it to list the value flags", err)
	}
}

func TestRunSetUnknownProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunSet(t.Context(), file, "nope", config.ProfileEdits{URL: ptr("https://x")}, nil, &bytes.Buffer{})
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Errorf("RunSet error = %v, want ErrProfileNotFound", err)
	}
}

// TestRunSetVerifiesMergedProfile checks the credentials that are verified are
// the ones that will be stored, not just the fields that changed.
func TestRunSetVerifiesMergedProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	verifier := &stubVerifier{identity: "someone@example.com"}
	edits := config.ProfileEdits{APIKey: ptr("mds_rotated123456")}
	if err := config.RunSet(t.Context(), file, "staging", edits, verifier, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSet: %v", err)
	}

	if verifier.got.OrganizationID != "acme" || verifier.got.APIKey != "mds_rotated123456" {
		t.Errorf("verified %+v, want the merged profile", verifier.got)
	}
}

func TestRunSetDoesNotWriteWhenVerificationFails(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	verifier := &stubVerifier{err: errors.New("unauthorized")}
	edits := config.ProfileEdits{APIKey: ptr("mds_rotated123456")}
	if err := config.RunSet(t.Context(), file, "staging", edits, verifier, &bytes.Buffer{}); err == nil {
		t.Fatal("RunSet succeeded despite failed verification")
	}

	saved, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if saved.Profiles["staging"].APIKey != "mds_stagingkey1234" {
		t.Error("profile was modified despite failed verification")
	}
}
