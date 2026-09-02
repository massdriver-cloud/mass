package resourcetype //nolint:testpackage // needs access to unexported packageKeep/validateReferencedFiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rtype "github.com/massdriver-cloud/mass/internal/resourcetype"
)

func TestPackageKeep(t *testing.T) {
	config := &rtype.MassdriverYAML{
		UI: &rtype.UIConfig{
			Instructions: []rtype.InstructionConfig{
				{Label: "CLI", Path: "./docs/cli.md"},
				{Label: "Console", Path: "instructions/console.md"},
			},
		},
		Exports: []rtype.ExportConfig{
			{DownloadButtonText: "Config", TemplatePath: "./templates/config.yaml.liquid"},
		},
	}
	keep := packageKeep(config)

	admit := []string{
		"massdriver.yaml",
		"README.md",
		"readme.md",
		"CHANGELOG.md",
		"icon.svg",
		"icon.png",
		"icon.jpg",
		"icon.jpeg",
		"docs/cli.md",                  // referenced instruction, arbitrary dir
		"instructions/console.md",      // referenced instruction
		"templates/config.yaml.liquid", // referenced export template
	}
	skip := []string{
		"main.tf",
		"schema-params.json",
		"icon.gif",
		".mdignore",
		"docs/other.md",       // unreferenced file in a referenced dir
		"instructions/cli.md", // not the referenced instruction path
		"secrets/key.pem",
	}

	for _, f := range admit {
		if !keep(f) {
			t.Errorf("keep(%q) = false, want true", f)
		}
	}
	for _, f := range skip {
		if keep(f) {
			t.Errorf("keep(%q) = true, want false", f)
		}
	}
}

func TestPackageKeepNoReferences(t *testing.T) {
	keep := packageKeep(&rtype.MassdriverYAML{})
	if !keep("massdriver.yaml") {
		t.Error("massdriver.yaml should always be kept")
	}
	if keep("instructions/cli.md") {
		t.Error("nothing under instructions/ should be kept when unreferenced")
	}
}

func TestValidateReferencedFiles(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(srcDir, "docs"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "docs", "cli.md"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "tmpl.liquid"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	uiWith := func(path string) *rtype.UIConfig {
		return &rtype.UIConfig{Instructions: []rtype.InstructionConfig{{Label: "L", Path: path}}}
	}

	t.Run("all references present and inside the tree", func(t *testing.T) {
		config := &rtype.MassdriverYAML{
			UI:      uiWith("./docs/cli.md"),
			Exports: []rtype.ExportConfig{{TemplatePath: "tmpl.liquid"}},
		}
		if err := validateReferencedFiles(config, srcDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing file is rejected", func(t *testing.T) {
		err := validateReferencedFiles(&rtype.MassdriverYAML{UI: uiWith("./docs/missing.md")}, srcDir)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("want not-found error, got: %v", err)
		}
	})

	t.Run("path escaping the directory is rejected", func(t *testing.T) {
		err := validateReferencedFiles(&rtype.MassdriverYAML{UI: uiWith("../secret.md")}, srcDir)
		if err == nil || !strings.Contains(err.Error(), "inside the resource type directory") {
			t.Fatalf("want outside-directory error, got: %v", err)
		}
	})

	t.Run("absolute path is rejected", func(t *testing.T) {
		err := validateReferencedFiles(&rtype.MassdriverYAML{Exports: []rtype.ExportConfig{{TemplatePath: "/etc/passwd"}}}, srcDir)
		if err == nil || !strings.Contains(err.Error(), "inside the resource type directory") {
			t.Fatalf("want outside-directory error, got: %v", err)
		}
	})

	t.Run("directory reference is rejected", func(t *testing.T) {
		err := validateReferencedFiles(&rtype.MassdriverYAML{UI: uiWith("./docs")}, srcDir)
		if err == nil || !strings.Contains(err.Error(), "is a directory") {
			t.Fatalf("want is-a-directory error, got: %v", err)
		}
	})
}
