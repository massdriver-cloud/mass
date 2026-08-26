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

// TestRunConvertRoundTrip converts a realistic raw schema and rebuilds it with
// resourcetype.Build, verifying that instruction/export content is extracted and
// restored and that numeric constraints survive (a regression guard for the
// json-float64 corruption that turned integers into scientific notation).
func TestRunConvertRoundTrip(t *testing.T) {
	dir := t.TempDir()
	raw := `{
  "$schema": "http://json-schema.org/draft-07/schema",
  "$md": {
    "name": "roundtrip",
    "label": "Round Trip",
    "icon": "https://example.com/icon.svg",
    "ui": {
      "connectionOrientation": "environmentDefault",
      "instructions": [
        { "label": "CLI Setup", "content": "step one\nstep two" }
      ]
    },
    "export": [
      { "downloadButtonText": "Download", "fileFormat": "yaml", "template": "key: {{ .val }}", "templateLang": "liquid" }
    ]
  },
  "type": "object",
  "required": ["token"],
  "properties": {
    "token": { "type": "string" },
    "count": { "type": "integer", "minimum": 2, "default": 1000000 }
  }
}`
	schemaPath := filepath.Join(dir, "raw.json")
	if err := os.WriteFile(schemaPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bundle", "massdriver.yaml")

	if _, err := cmdresourcetype.RunConvert(schemaPath, out, false); err != nil {
		t.Fatalf("RunConvert failed: %v", err)
	}

	built, err := rtype.Build(out)
	if err != nil {
		t.Fatalf("rebuilding the converted massdriver.yaml failed: %v", err)
	}

	md, ok := built["$md"].(map[string]any)
	if !ok {
		t.Fatalf("$md missing from rebuilt schema: %#v", built)
	}
	if md["name"] != "roundtrip" {
		t.Errorf("name = %v, want roundtrip", md["name"])
	}

	// Instruction content extracted to a file and restored on rebuild.
	ui, _ := md["ui"].(map[string]any)
	instructions, _ := ui["instructions"].([]map[string]any)
	if len(instructions) != 1 || instructions[0]["content"] != "step one\nstep two" {
		t.Errorf("instruction content not restored: %#v", instructions)
	}

	// Export template extracted to a file and restored on rebuild.
	exports, _ := md["export"].([]map[string]any)
	if len(exports) != 1 || exports[0]["template"] != "key: {{ .val }}" {
		t.Errorf("export template not restored: %#v", exports)
	}

	// Numeric fidelity: the large integer default must round-trip as an int,
	// not a float rendered in scientific notation.
	props, _ := built["properties"].(map[string]any)
	count, _ := props["count"].(map[string]any)
	if d, ok := count["default"].(int); !ok || d != 1000000 {
		t.Errorf("count.default = %#v (%T), want int 1000000", count["default"], count["default"])
	}
}

func TestRunConvertYAMLInput(t *testing.T) {
	dir := t.TempDir()
	raw := "$md:\n  name: from-yaml\ntype: object\nproperties:\n  token:\n    type: string\n"
	schemaPath := filepath.Join(dir, "raw.yaml")
	if err := os.WriteFile(schemaPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "massdriver.yaml")

	if _, err := cmdresourcetype.RunConvert(schemaPath, out, false); err != nil {
		t.Fatalf("RunConvert failed for YAML input: %v", err)
	}

	data, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var config rtype.MassdriverYAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("output is not valid massdriver.yaml: %v", err)
	}
	if config.Name != "from-yaml" {
		t.Errorf("name = %q, want from-yaml", config.Name)
	}
	if _, ok := config.Schema["properties"]; !ok {
		t.Error("schema should retain properties from YAML input")
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
