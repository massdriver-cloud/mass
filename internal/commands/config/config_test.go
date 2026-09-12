package config_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// loadConfig points config resolution at a temp dir holding contents, clears
// the environment variables that would otherwise leak the developer's own
// credentials into a test, and returns the loaded file.
func loadConfig(t *testing.T, contents string) *configfile.File {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MASSDRIVER_PROFILE", "")
	t.Setenv("MASSDRIVER_ORGANIZATION_ID", "")
	t.Setenv("MASSDRIVER_ORG_ID", "")
	t.Setenv("MASSDRIVER_API_KEY", "")
	t.Setenv("MASSDRIVER_URL", "")
	t.Setenv("MASSDRIVER_TEMPLATES_PATH", "")

	if contents != "" {
		path := filepath.Join(dir, "massdriver", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("configfile.Load: %v", err)
	}
	return file
}

// twoProfiles is the fixture most tests read: a default and a staging profile
// with staging active.
const twoProfiles = `version: 1
current_profile: staging
profiles:
  default:
    organization_id: acme
    api_key: mds_defaultkey1234
  staging:
    organization_id: acme
    api_key: mds_stagingkey1234
    url: https://api.massdriver.example.com
`

func ptr(s string) *string { return &s }

// stubVerifier records what it was asked to check so tests can assert the
// credentials are verified before they reach disk.
type stubVerifier struct {
	identity string
	err      error
	got      sdkconfig.Profile
	calls    int
}

func (s *stubVerifier) Verify(_ context.Context, p sdkconfig.Profile) (string, error) {
	s.calls++
	s.got = p
	return s.identity, s.err
}

// stubPrompter answers prompts from a queue.
type stubPrompter struct {
	answers []string
	labels  []string
}

func (s *stubPrompter) Prompt(label string, _ bool) (string, error) {
	s.labels = append(s.labels, label)
	if len(s.answers) == 0 {
		return "", errors.New("no answer queued")
	}
	answer := s.answers[0]
	s.answers = s.answers[1:]
	return answer, nil
}

func TestProfileEditsApply(t *testing.T) {
	base := sdkconfig.Profile{
		OrganizationID: "acme",
		APIKey:         "mds_old",
		URL:            "https://api.massdriver.example.com",
		TemplatesPath:  "/tmp/templates",
	}

	tests := []struct {
		name  string
		edits config.ProfileEdits
		want  sdkconfig.Profile
	}{
		{
			name:  "no edits leaves the profile alone",
			edits: config.ProfileEdits{},
			want:  base,
		},
		{
			name:  "one edit touches only that field",
			edits: config.ProfileEdits{APIKey: ptr("mds_new")},
			want: sdkconfig.Profile{
				OrganizationID: "acme",
				APIKey:         "mds_new",
				URL:            "https://api.massdriver.example.com",
				TemplatesPath:  "/tmp/templates",
			},
		},
		{
			name:  "an explicit empty string clears a field",
			edits: config.ProfileEdits{TemplatesPath: ptr("")},
			want: sdkconfig.Profile{
				OrganizationID: "acme",
				APIKey:         "mds_old",
				URL:            "https://api.massdriver.example.com",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := base
			tc.edits.Apply(&got)
			if got != tc.want {
				t.Errorf("Apply() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestProfileEditsAny(t *testing.T) {
	if (config.ProfileEdits{}).Any() {
		t.Error("Any() = true for empty edits")
	}
	if !(config.ProfileEdits{URL: ptr("")}).Any() {
		t.Error("Any() = false for an explicitly cleared field")
	}
}
