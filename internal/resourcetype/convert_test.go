package resourcetype_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"gopkg.in/yaml.v3"
)

func TestConvert(t *testing.T) {
	out := filepath.Join(t.TempDir(), "massdriver.yaml")

	result, err := resourcetype.Convert("testdata/simple-resource.json", out, false)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if result.MassdriverYAML != out {
		t.Errorf("MassdriverYAML = %q, want %q", result.MassdriverYAML, out)
	}

	data, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatalf("reading output: %v", readErr)
	}

	var config resourcetype.MassdriverYAML
	if unmarshalErr := yaml.Unmarshal(data, &config); unmarshalErr != nil {
		t.Fatalf("output is not valid massdriver.yaml: %v", unmarshalErr)
	}

	if config.Name != "foo" {
		t.Errorf("name = %q, want %q", config.Name, "foo")
	}
	if config.Version == "" {
		t.Error("expected a placeholder version to be written")
	}
	// The $md block must be lifted out of the schema.
	if _, ok := config.Schema["$md"]; ok {
		t.Error("schema should not contain the $md block after conversion")
	}
	if _, ok := config.Schema["properties"]; !ok {
		t.Error("schema should retain the original JSON schema keys (properties)")
	}
}

func TestConvertDistinctFilesForDuplicateLabels(t *testing.T) {
	dir := t.TempDir()
	raw := `{
  "$md": {
    "name": "dup",
    "ui": { "instructions": [
      { "label": "Setup", "content": "first" },
      { "label": "Setup", "content": "second" }
    ] }
  },
  "type": "object"
}`
	schemaPath := filepath.Join(dir, "raw.json")
	if err := os.WriteFile(schemaPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", "massdriver.yaml")

	result, err := resourcetype.Convert(schemaPath, out, false)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(result.ExtraFiles) != 2 {
		t.Fatalf("expected 2 distinct instruction files, got %d: %v", len(result.ExtraFiles), result.ExtraFiles)
	}

	contents := map[string]bool{}
	for _, f := range result.ExtraFiles {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatal(readErr)
		}
		contents[string(data)] = true
	}
	if !contents["first"] || !contents["second"] {
		t.Errorf("both instruction contents should be preserved, got: %v", contents)
	}
}

func TestConvertRefusesToClobber(t *testing.T) {
	out := filepath.Join(t.TempDir(), "massdriver.yaml")
	if err := os.WriteFile(out, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := resourcetype.Convert("testdata/simple-resource.json", out, false); err == nil {
		t.Fatal("expected an error when the output file already exists")
	}

	if _, err := resourcetype.Convert("testdata/simple-resource.json", out, true); err != nil {
		t.Fatalf("expected --force to overwrite, got: %v", err)
	}
}
