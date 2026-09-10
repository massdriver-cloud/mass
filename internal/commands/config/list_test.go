package config_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/config"
)

func TestRunListMarksActiveProfile(t *testing.T) {
	tests := []struct {
		name     string
		override string
		want     string
	}{
		{name: "marks current_profile", want: "staging"},
		{name: "the --profile override moves the marker", override: "default", want: "default"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := loadConfig(t, twoProfiles)

			var out bytes.Buffer
			opts := config.ListOptions{Output: "json", ProfileOverride: tc.override}
			if err := config.RunList(file, opts, &out); err != nil {
				t.Fatalf("RunList: %v", err)
			}

			var rendered []struct {
				Name   string `json:"name"`
				Active bool   `json:"active"`
			}
			if err := json.Unmarshal(out.Bytes(), &rendered); err != nil {
				t.Fatalf("Unmarshal: %v\n%s", err, out.String())
			}
			if len(rendered) != 2 {
				t.Fatalf("listed %d profiles, want 2", len(rendered))
			}
			for _, p := range rendered {
				if p.Active != (p.Name == tc.want) {
					t.Errorf("profile %q active = %v, want %v", p.Name, p.Active, p.Name == tc.want)
				}
			}
		})
	}
}

func TestRunListMasksAPIKeys(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var masked bytes.Buffer
	if err := config.RunList(file, config.ListOptions{Output: "table"}, &masked); err != nil {
		t.Fatalf("RunList: %v", err)
	}
	if strings.Contains(masked.String(), "mds_stagingkey1234") {
		t.Errorf("API key printed in full by default:\n%s", masked.String())
	}
	if !strings.Contains(masked.String(), "****1234") {
		t.Errorf("masked key not shown:\n%s", masked.String())
	}

	var shown bytes.Buffer
	if err := config.RunList(file, config.ListOptions{Output: "table", ShowSecrets: true}, &shown); err != nil {
		t.Fatalf("RunList: %v", err)
	}
	if !strings.Contains(shown.String(), "mds_stagingkey1234") {
		t.Errorf("--show-secrets did not print the key:\n%s", shown.String())
	}
}

// TestRunListHeaderNamesTheFileNotTheProfile guards against the header reading
// as though the config path were the active profile's name.
func TestRunListHeaderNamesTheFileNotTheProfile(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	var out bytes.Buffer
	if err := config.RunList(file, config.ListOptions{Output: "table"}, &out); err != nil {
		t.Fatalf("RunList: %v", err)
	}

	header, _, found := strings.Cut(out.String(), "\n")
	if !found {
		t.Fatalf("output has no header line:\n%s", out.String())
	}
	if !strings.Contains(header, file.Path()) {
		t.Errorf("header = %q, want it to name the config file", header)
	}
	if !strings.Contains(header, "* = active") {
		t.Errorf("header = %q, want it to explain the * marker", header)
	}
	if strings.Contains(header, "active profile") {
		t.Errorf("header = %q reads as though the path were the active profile", header)
	}
}

func TestRunListWithNoProfiles(t *testing.T) {
	file := loadConfig(t, "")

	var out bytes.Buffer
	if err := config.RunList(file, config.ListOptions{Output: "table"}, &out); err != nil {
		t.Fatalf("RunList: %v", err)
	}
	if !strings.Contains(out.String(), "No profiles configured") {
		t.Errorf("output = %q, want a message about having no profiles", out.String())
	}
}

func TestRunListRejectsUnknownOutputFormat(t *testing.T) {
	file := loadConfig(t, twoProfiles)

	err := config.RunList(file, config.ListOptions{Output: "yaml"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("RunList with an unsupported format succeeded, want an error")
	}
}
