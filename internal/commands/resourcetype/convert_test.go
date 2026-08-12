package resourcetype_test

import (
	"os"
	"path/filepath"
	"testing"

	cmdresourcetype "github.com/massdriver-cloud/mass/internal/commands/resourcetype"
	rtype "github.com/massdriver-cloud/mass/internal/resourcetype"
	"gopkg.in/yaml.v3"
)

func TestRunConvert(t *testing.T) {
	out := filepath.Join(t.TempDir(), "massdriver.yaml")

	result, err := cmdresourcetype.RunConvert("testdata/simple-resource.json", out, false)
	if err != nil {
		t.Fatalf("RunConvert failed: %v", err)
	}
	if result.MassdriverYAML != out {
		t.Errorf("MassdriverYAML = %q, want %q", result.MassdriverYAML, out)
	}

	data, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatalf("reading output: %v", readErr)
	}

	var config rtype.MassdriverYAML
	if unmarshalErr := yaml.Unmarshal(data, &config); unmarshalErr != nil {
		t.Fatalf("output is not valid massdriver.yaml: %v", unmarshalErr)
	}

	if config.Name != "foo" {
		t.Errorf("name = %q, want %q", config.Name, "foo")
	}
	if config.Version == "" {
		t.Error("expected a placeholder version to be written")
	}
	if _, ok := config.Schema["$md"]; ok {
		t.Error("schema should not contain the $md block after conversion")
	}
	if _, ok := config.Schema["properties"]; !ok {
		t.Error("schema should retain the original JSON schema keys (properties)")
	}
}

func TestRunConvertDistinctFilesForDuplicateLabels(t *testing.T) {
	dir := t.TempDir()
	// Labels crafted to trip the old (buggy) unique-path logic: the third
	// instruction's fallback name collided with the first's.
	raw := `{
  "$md": {
    "name": "dup",
    "ui": { "instructions": [
      { "label": "a 3", "content": "first" },
      { "label": "a", "content": "second" },
      { "label": "a", "content": "third" }
    ] }
  },
  "type": "object"
}`
	schemaPath := filepath.Join(dir, "raw.json")
	if err := os.WriteFile(schemaPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", "massdriver.yaml")

	result, err := cmdresourcetype.RunConvert(schemaPath, out, false)
	if err != nil {
		t.Fatalf("RunConvert failed: %v", err)
	}
	if len(result.ExtraFiles) != 3 {
		t.Fatalf("expected 3 distinct instruction files, got %d: %v", len(result.ExtraFiles), result.ExtraFiles)
	}

	contents := map[string]bool{}
	for _, f := range result.ExtraFiles {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatal(readErr)
		}
		contents[string(data)] = true
	}
	for _, want := range []string{"first", "second", "third"} {
		if !contents[want] {
			t.Errorf("instruction content %q was lost to a filename collision, got: %v", want, contents)
		}
	}
}

func TestRunConvertRefusesToClobber(t *testing.T) {
	out := filepath.Join(t.TempDir(), "massdriver.yaml")
	if err := os.WriteFile(out, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := cmdresourcetype.RunConvert("testdata/simple-resource.json", out, false); err == nil {
		t.Fatal("expected an error when the output file already exists")
	}

	if _, err := cmdresourcetype.RunConvert("testdata/simple-resource.json", out, true); err != nil {
		t.Fatalf("expected --force to overwrite, got: %v", err)
	}
}
