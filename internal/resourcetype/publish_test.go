package resourcetype_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/resourcetype"
)

// TestPublishValidation covers the local validation Publish performs before it
// touches the OCI registry: rejecting raw schema files (pointing at convert)
// and requiring name/version in the massdriver.yaml. These paths short-circuit
// before the massdriver client is used, so a nil client is fine.
func TestPublishValidation(t *testing.T) {
	dir := t.TempDir()

	rawJSON := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(rawJSON, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	noVersionDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(noVersionDir, "massdriver.yaml"), []byte("name: foo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	noNameDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(noNameDir, "massdriver.yaml"), []byte("version: 1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	emptyDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{name: "raw JSON schema rejected", path: rawJSON, contains: "convert"},
		{name: "directory without massdriver.yaml", path: emptyDir, contains: "no massdriver.yaml"},
		{name: "missing version", path: noVersionDir, contains: "version is required"},
		{name: "missing name", path: noNameDir, contains: "name is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := resourcetype.Publish(t.Context(), nil, tc.path)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("expected error to contain %q, got: %v", tc.contains, err)
			}
		})
	}
}
