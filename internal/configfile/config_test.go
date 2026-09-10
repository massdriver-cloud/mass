package configfile_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// withConfigDir points config resolution at a temp dir and clears the
// environment variables that would otherwise leak the developer's own
// credentials into a test.
func withConfigDir(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MASSDRIVER_PROFILE", "")
	t.Setenv("MASSDRIVER_ORGANIZATION_ID", "")
	t.Setenv("MASSDRIVER_ORG_ID", "")
	t.Setenv("MASSDRIVER_API_KEY", "")
	t.Setenv("MASSDRIVER_URL", "")
	t.Setenv("MASSDRIVER_TEMPLATES_PATH", "")

	path := filepath.Join(dir, "massdriver", "config.yaml")
	if contents != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return path
}

func TestPathMatchesSDK(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got, err := configfile.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want, err := sdkconfig.FilePath()
	if err != nil {
		t.Fatalf("sdkconfig.FilePath: %v", err)
	}
	if got != want {
		t.Errorf("Path() = %q, want %q (the SDK's location)", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	withConfigDir(t, "")

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if file.Exists() {
		t.Error("Exists() = true for a file that was never created")
	}
	if got := file.Profiles(); len(got) != 0 {
		t.Errorf("Profiles() = %v, want empty", got)
	}
	if got := file.ActiveProfileName(""); got != sdkconfig.DefaultProfileName {
		t.Errorf("ActiveProfileName() = %q, want %q", got, sdkconfig.DefaultProfileName)
	}
}

// TestLoadRefusesFileTheSDKRejects keeps the CLI from editing a file the SDK
// cannot read, which would otherwise let `mass config` write into a file no
// other command can load.
func TestLoadRefusesFileTheSDKRejects(t *testing.T) {
	withConfigDir(t, "version: 2\nprofiles: {}\n")

	_, err := configfile.Load()
	if err == nil {
		t.Fatal("Load() succeeded, want the SDK's unsupported-version error")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("Load() error = %v, want it to mention the version", err)
	}
}

func TestProfilesAreSortedByName(t *testing.T) {
	withConfigDir(t, `version: 1
profiles:
  staging:
    organization_id: acme
    api_key: mds_staging
  default:
    organization_id: acme
    api_key: mds_default
    url: https://api.massdriver.example.com
    templates_path: /tmp/templates
`)

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	profiles := file.Profiles()
	if len(profiles) != 2 {
		t.Fatalf("Profiles() returned %d profiles, want 2", len(profiles))
	}
	if profiles[0].Name != "default" || profiles[1].Name != "staging" {
		t.Errorf("Profiles() names = %q, %q, want default, staging", profiles[0].Name, profiles[1].Name)
	}

	want := sdkconfig.Profile{
		OrganizationID: "acme",
		APIKey:         "mds_default",
		URL:            "https://api.massdriver.example.com",
		TemplatesPath:  "/tmp/templates",
	}
	if profiles[0].Profile != want {
		t.Errorf("Profiles()[0] = %+v, want %+v", profiles[0].Profile, want)
	}
}

func TestProfileNotFound(t *testing.T) {
	withConfigDir(t, "version: 1\nprofiles:\n  default:\n    organization_id: acme\n")

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, found := file.Profile("nope"); found {
		t.Error("Profile(\"nope\") reported found")
	}
}

// TestSetProfilePreservesComments is the reason edits go through yaml.Node:
// users have been hand-editing this file, and an update must not wipe their
// annotations, key ordering, or keys this CLI doesn't model.
func TestSetProfilePreservesComments(t *testing.T) {
	path := withConfigDir(t, `# Massdriver CLI configuration
version: 1
profiles:
  # our shared staging org
  staging:
    organization_id: acme
    api_key: mds_old # rotate quarterly
    custom_key: keep-me
`)

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	profile, found := file.Profile("staging")
	if !found {
		t.Fatal("Profile(\"staging\") not found")
	}
	profile.APIKey = "mds_new"
	if setErr := file.SetProfile("staging", profile); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	for _, want := range []string{
		"# Massdriver CLI configuration",
		"# our shared staging org",
		"# rotate quarterly",
		"custom_key: keep-me",
		"mds_new",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("saved file is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "mds_old") {
		t.Errorf("saved file still contains the old API key:\n%s", got)
	}
}

func TestSetProfileRemovesEmptyFields(t *testing.T) {
	path := withConfigDir(t, `version: 1
profiles:
  staging:
    organization_id: acme
    api_key: mds_key
    templates_path: /tmp/templates
`)

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	profile, _ := file.Profile("staging")
	profile.TemplatesPath = ""
	if setErr := file.SetProfile("staging", profile); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "templates_path") {
		t.Errorf("cleared field was written to the file:\n%s", data)
	}
}

func TestSetProfileCreatesFileWithVersion(t *testing.T) {
	withConfigDir(t, "")

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if setErr := file.SetProfile("default", sdkconfig.Profile{OrganizationID: "acme", APIKey: "mds_key"}); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	// Reading back through the SDK proves the version we stamped is the one
	// the SDK accepts.
	reloaded, err := sdkconfig.ReadFile()
	if err != nil {
		t.Fatalf("SDK could not read the file we created: %v", err)
	}
	profile, found := reloaded.Profiles["default"]
	if !found {
		t.Fatal("profile was not persisted")
	}
	if profile.OrganizationID != "acme" || profile.APIKey != "mds_key" {
		t.Errorf("reloaded profile = %+v", profile)
	}
}

func TestSetProfileRequiresName(t *testing.T) {
	withConfigDir(t, "")

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = file.SetProfile("", sdkconfig.Profile{OrganizationID: "acme"}); err == nil {
		t.Error("SetProfile with an empty name succeeded, want an error")
	}
}

// TestSaveIsOwnerOnly guards the file that holds API keys.
func TestSaveIsOwnerOnly(t *testing.T) {
	path := withConfigDir(t, "version: 1\nprofiles:\n  default:\n    organization_id: acme\n")

	if err := os.Chmod(path, 0644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if setErr := file.SetProfile("default", sdkconfig.Profile{OrganizationID: "acme", APIKey: "mds_key"}); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file mode = %04o, want 0600", perm)
	}
}

// TestRemoveProfileClearsCurrentProfile matters more now that the SDK treats
// current_profile as an explicit selection: leaving it pointed at a deleted
// profile would fail every subsequent command with ErrProfileNotFound.
func TestRemoveProfileClearsCurrentProfile(t *testing.T) {
	path := withConfigDir(t, `version: 1
current_profile: staging
profiles:
  default:
    organization_id: acme
    api_key: mds_default
  staging:
    organization_id: acme
    api_key: mds_staging
`)

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !file.RemoveProfile("staging") {
		t.Fatal("RemoveProfile(\"staging\") = false, want true")
	}
	if file.RemoveProfile("staging") {
		t.Error("RemoveProfile on an already-removed profile = true, want false")
	}
	if current := file.CurrentProfile(); current != "" {
		t.Errorf("CurrentProfile() = %q after removing the active profile, want empty", current)
	}
	if _, found := file.Profile("default"); !found {
		t.Error("removing staging also removed default")
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	if data, readErr := os.ReadFile(path); readErr == nil && strings.Contains(string(data), "current_profile") {
		t.Errorf("current_profile still points at the removed profile:\n%s", data)
	}

	// The SDK must now resolve without error rather than tripping over a
	// dangling current_profile.
	if _, loadErr := sdkconfig.Get(); loadErr != nil {
		t.Errorf("SDK could not resolve after removing the active profile: %v", loadErr)
	}
}

// TestSDKHonorsCurrentProfileWeWrite is the contract this feature rests on:
// `mass config use` writes current_profile, and the SDK — with no override and
// no environment variable — selects it.
func TestSDKHonorsCurrentProfileWeWrite(t *testing.T) {
	withConfigDir(t, "")

	file, err := configfile.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if setErr := file.SetProfile("default", sdkconfig.Profile{OrganizationID: "other", APIKey: "mds_other"}); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if setErr := file.SetProfile("staging", sdkconfig.Profile{
		OrganizationID: "acme",
		APIKey:         "mds_key",
		URL:            "https://api.massdriver.example.com",
		TemplatesPath:  "/tmp/templates",
	}); setErr != nil {
		t.Fatalf("SetProfile: %v", setErr)
	}
	if setErr := file.SetCurrentProfile("staging"); setErr != nil {
		t.Fatalf("SetCurrentProfile: %v", setErr)
	}
	if saveErr := file.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	cfg, err := sdkconfig.Get()
	if err != nil {
		t.Fatalf("SDK could not load the config we wrote: %v", err)
	}
	if cfg.Profile != "staging" {
		t.Errorf("SDK selected profile %q, want staging (current_profile was ignored)", cfg.Profile)
	}
	if cfg.OrganizationID != "acme" {
		t.Errorf("SDK OrganizationID = %q, want acme", cfg.OrganizationID)
	}
	if cfg.URL != "https://api.massdriver.example.com" {
		t.Errorf("SDK URL = %q", cfg.URL)
	}
	if cfg.TemplatesPath != "/tmp/templates" {
		t.Errorf("SDK TemplatesPath = %q", cfg.TemplatesPath)
	}
	if cfg.Credentials.Source != sdkconfig.SourceProfile {
		t.Errorf("SDK credential source = %q, want profile", cfg.Credentials.Source)
	}
}

// TestActiveProfileNameMatchesSDK pins the label shown by `mass config list`
// and `mass config get` to the profile the SDK actually selects. The SDK's
// precedence is authoritative; this test is what keeps the display helper from
// drifting away from it.
func TestActiveProfileNameMatchesSDK(t *testing.T) {
	tests := []struct {
		name       string
		contents   string
		envProfile string
		override   string
		want       string
	}{
		{
			name: "falls back to the default profile",
			contents: `version: 1
profiles:
  default:
    organization_id: acme
    api_key: mds_default
`,
			want: "default",
		},
		{
			name: "honors current_profile",
			contents: `version: 1
current_profile: staging
profiles:
  default:
    organization_id: acme
    api_key: mds_default
  staging:
    organization_id: acme
    api_key: mds_staging
`,
			want: "staging",
		},
		{
			name: "MASSDRIVER_PROFILE beats current_profile",
			contents: `version: 1
current_profile: staging
profiles:
  staging:
    organization_id: acme
    api_key: mds_staging
  production:
    organization_id: acme
    api_key: mds_production
`,
			envProfile: "production",
			want:       "production",
		},
		{
			name: "the --profile override beats everything",
			contents: `version: 1
current_profile: staging
profiles:
  staging:
    organization_id: acme
    api_key: mds_staging
  production:
    organization_id: acme
    api_key: mds_production
`,
			envProfile: "staging",
			override:   "production",
			want:       "production",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withConfigDir(t, tc.contents)
			if tc.envProfile != "" {
				t.Setenv("MASSDRIVER_PROFILE", tc.envProfile)
			}

			file, err := configfile.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := file.ActiveProfileName(tc.override)
			if got != tc.want {
				t.Errorf("ActiveProfileName(%q) = %q, want %q", tc.override, got, tc.want)
			}

			// Overrides.Profile is what newMassdriverClient passes for
			// --profile, so this is the selection the CLI actually gets.
			cfg, err := sdkconfig.Load(sdkconfig.Overrides{Profile: tc.override})
			if err != nil {
				t.Fatalf("sdkconfig.Load: %v", err)
			}
			if cfg.Profile != got {
				t.Errorf("ActiveProfileName(%q) = %q but the SDK selected %q", tc.override, got, cfg.Profile)
			}
		})
	}
}

// TestSDKFailsOnDanglingCurrentProfile documents why `mass config use`
// validates the profile exists before writing it.
func TestSDKFailsOnDanglingCurrentProfile(t *testing.T) {
	withConfigDir(t, `version: 1
current_profile: gone
profiles:
  default:
    organization_id: acme
    api_key: mds_default
`)

	_, err := sdkconfig.Get()
	if !errors.Is(err, sdkconfig.ErrProfileNotFound) {
		t.Errorf("sdkconfig.Get() error = %v, want ErrProfileNotFound", err)
	}
}
