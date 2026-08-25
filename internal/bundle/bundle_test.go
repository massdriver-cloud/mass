package bundle //nolint:testpackage // exercises unexported normalizeInputs/dependencySchema

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

// TestUnmarshalDependencyResourceVariants covers the three ways `dependencies`
// and `resources` can be "empty": missing entirely, present-but-null, and an
// empty object. All must unmarshal cleanly to an empty dependency schema.
func TestUnmarshalDependencyResourceVariants(t *testing.T) {
	const base = "name: example\ndescription: a bundle\nversion: 1.0.0\nsteps:\n  - path: src\n    provisioner: terraform\nparams:\n  properties: {}\nui: {}\n"
	cases := map[string]string{
		"missing": base,
		"null":    base + "dependencies:\nresources:\n",
		"empty":   base + "dependencies: {}\nresources: {}\n",
	}

	for name, contents := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "massdriver.yaml"), []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}

			b, err := Unmarshal(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			props, ok := b.dependencySchema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("dependencySchema has no properties map: %#v", b.dependencySchema)
			}
			if len(props) != 0 {
				t.Errorf("expected empty dependency schema, got: %#v", props)
			}
		})
	}
}

func TestDependenciesToSchema(t *testing.T) {
	deps := map[string]Dependency{
		"network":  {ResourceType: "aws-vpc@1.2.3", Required: boolPtr(true)},
		"database": {ResourceType: "postgres@2.3.4", Required: boolPtr(false)},
	}

	got := dependenciesToSchema(deps)

	want := map[string]any{
		"properties": map[string]any{
			"network":  map[string]any{"$ref": "aws-vpc@1.2.3"},
			"database": map[string]any{"$ref": "postgres@2.3.4"},
		},
		"required": []any{"network"}, // only required==true, sorted
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestNormalizeInputs(t *testing.T) {
	t.Run("dependencies hydrate dependencySchema without touching Connections", func(t *testing.T) {
		b := &Bundle{
			Version:      "1.0.0",
			Dependencies: map[string]Dependency{"network": {ResourceType: "aws-vpc@1.2.3", Required: boolPtr(true)}},
		}
		if err := b.normalizeInputs(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Connections != nil {
			t.Errorf("Connections should remain untouched, got: %#v", b.Connections)
		}
		props, ok := b.dependencySchema["properties"].(map[string]any)
		if !ok || props["network"] == nil {
			t.Fatalf("dependencySchema not hydrated from dependencies: %#v", b.dependencySchema)
		}
	})

	t.Run("resources stay first-class without touching Artifacts", func(t *testing.T) {
		b := &Bundle{
			Version:   "1.0.0",
			Resources: map[string]Resource{"bucket": {ResourceType: "aws-s3@1.0.0", Required: boolPtr(true)}},
		}
		if err := b.normalizeInputs(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Artifacts != nil {
			t.Errorf("Artifacts should remain untouched, got: %#v", b.Artifacts)
		}
		if b.Resources == nil {
			t.Error("Resources should remain the first-class field")
		}
	})

	t.Run("legacy connections hydrate dependencySchema at 0.0.0", func(t *testing.T) {
		b := &Bundle{Version: "0.0.0", Connections: map[string]any{"properties": map[string]any{"legacy": map[string]any{"$ref": "aws-vpc"}}}}
		if err := b.normalizeInputs(); err != nil {
			t.Fatalf("legacy term should be allowed at 0.0.0, got: %v", err)
		}
		props, ok := b.dependencySchema["properties"].(map[string]any)
		if !ok || props["legacy"] == nil {
			t.Fatalf("dependencySchema not hydrated from legacy connections: %#v", b.dependencySchema)
		}
	})

	t.Run("legacy connections at a real version are rejected", func(t *testing.T) {
		b := &Bundle{Version: "1.0.0", Connections: map[string]any{"properties": map[string]any{}}}
		err := b.normalizeInputs()
		if err == nil || !strings.Contains(err.Error(), "deprecated") {
			t.Fatalf("want deprecation error at real version, got: %v", err)
		}
	})

	t.Run("legacy artifacts at a real version are rejected", func(t *testing.T) {
		b := &Bundle{Version: "2.1.0", Artifacts: map[string]any{"properties": map[string]any{}}}
		err := b.normalizeInputs()
		if err == nil || !strings.Contains(err.Error(), "deprecated") {
			t.Fatalf("want deprecation error at real version, got: %v", err)
		}
	})

	t.Run("both connections and dependencies is an error", func(t *testing.T) {
		b := &Bundle{
			Version:      "0.0.0",
			Connections:  map[string]any{},
			Dependencies: map[string]Dependency{"network": {ResourceType: "aws-vpc", Required: boolPtr(true)}},
		}
		err := b.normalizeInputs()
		if err == nil || !strings.Contains(err.Error(), "both") {
			t.Fatalf("want both-set error, got: %v", err)
		}
	})

	t.Run("both artifacts and resources is an error", func(t *testing.T) {
		b := &Bundle{
			Version:   "0.0.0",
			Artifacts: map[string]any{},
			Resources: map[string]Resource{"bucket": {ResourceType: "aws-s3", Required: boolPtr(true)}},
		}
		err := b.normalizeInputs()
		if err == nil || !strings.Contains(err.Error(), "both") {
			t.Fatalf("want both-set error, got: %v", err)
		}
	})
}
