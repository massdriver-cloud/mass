package resourcetype //nolint:testpackage // needs access to unexported resolvePublishPath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolvePublishPath pins which publish argument shapes take the OCI flow
// and which fall back to the deprecated raw-schema flow. Getting this wrong
// either breaks customers still publishing JSON schemas or silently pushes a
// massdriver.yaml through the unversioned legacy mutation.
func TestResolvePublishPath(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	mdYamlDir := filepath.Join(dir, "rt")
	mdYaml := write("rt/massdriver.yaml")
	jsonSchema := write("schema.json")
	yamlSchema := write("schema.yaml")
	ymlSchema := write("schema.yml")
	upperJSON := write("Schema.JSON")
	unsupported := write("schema.txt")

	tests := []struct {
		name       string
		path       string
		wantPath   string
		wantSrcDir string
		wantLegacy bool
		wantErr    string
	}{
		{name: "directory with massdriver.yaml", path: mdYamlDir, wantPath: mdYaml, wantSrcDir: mdYamlDir},
		{name: "massdriver.yaml file", path: mdYaml, wantPath: mdYaml, wantSrcDir: mdYamlDir},
		{name: "json schema is legacy", path: jsonSchema, wantPath: jsonSchema, wantLegacy: true},
		{name: "yaml schema is legacy", path: yamlSchema, wantPath: yamlSchema, wantLegacy: true},
		{name: "yml schema is legacy", path: ymlSchema, wantPath: ymlSchema, wantLegacy: true},
		{name: "extension match is case-insensitive", path: upperJSON, wantPath: upperJSON, wantLegacy: true},
		{name: "unsupported extension", path: unsupported, wantErr: "unsupported resource type path"},
		{name: "directory without massdriver.yaml", path: t.TempDir(), wantErr: "no massdriver.yaml"},
		{name: "missing path", path: filepath.Join(dir, "nope"), wantErr: "failed to read resource type path"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePublishPath(tc.path)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q missing %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePublishPath returned an error: %v", err)
			}
			if got.path != tc.wantPath {
				t.Errorf("path = %q, want %q", got.path, tc.wantPath)
			}
			if got.srcDir != tc.wantSrcDir {
				t.Errorf("srcDir = %q, want %q", got.srcDir, tc.wantSrcDir)
			}
			if got.legacy != tc.wantLegacy {
				t.Errorf("legacy = %v, want %v", got.legacy, tc.wantLegacy)
			}
		})
	}
}
