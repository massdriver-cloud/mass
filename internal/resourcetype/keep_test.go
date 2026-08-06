package resourcetype //nolint:testpackage // needs access to unexported packageKeep

import "testing"

func TestPackageKeep(t *testing.T) {
	keep := []string{
		"massdriver.yaml",
		"README.md",
		"readme.md",
		"CHANGELOG.md",
		"icon.svg",
		"icon.png",
		"icon.jpg",
		"icon.jpeg",
		"instructions/cli.md",
		"instructions/nested/deep.md",
		"exports/config.yaml.liquid",
	}
	drop := []string{
		"main.tf",
		"schema-params.json",
		"icon.gif",
		".mdignore",
		"instructions", // the bare name, not a file under the dir
		"docs/readme.md",
		"secrets/key.pem",
	}

	for _, f := range keep {
		if !packageKeep(f) {
			t.Errorf("packageKeep(%q) = false, want true", f)
		}
	}
	for _, f := range drop {
		if packageKeep(f) {
			t.Errorf("packageKeep(%q) = true, want false", f)
		}
	}
}
