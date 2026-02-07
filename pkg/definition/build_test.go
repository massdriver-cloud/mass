package definition_test

import (
	"path/filepath"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/definition"
)

func TestIsMassdriverYAMLArtifactDefinition(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "massdriver.yaml file",
			path: "some/path/massdriver.yaml",
			want: true,
		},
		{
			name: "massdriver.yaml in root",
			path: "massdriver.yaml",
			want: true,
		},
		{
			name: "json artifact definition",
			path: "some/path/artifact.json",
			want: false,
		},
		{
			name: "other yaml file",
			path: "some/path/artifact.yaml",
			want: false,
		},
		{
			name: "yml file",
			path: "some/path/artifact.yml",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := definition.IsMassdriverYAMLArtifactDefinition(tc.path)
			if got != tc.want {
				t.Errorf("IsMassdriverYAMLArtifactDefinition(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(t *testing.T, result map[string]any)
	}{
		{
			name:    "builds massdriver.yaml with instructions and exports",
			path:    filepath.Join("testdata", "massdriver-yaml-artifact", "massdriver.yaml"),
			wantErr: false,
			check: func(t *testing.T, result map[string]any) {
				// Check $md block
				md, ok := result["$md"].(map[string]any)
				if !ok {
					t.Fatal("expected $md to be a map")
				}

				if md["name"] != "test-artifact" {
					t.Errorf("expected name to be test-artifact, got %v", md["name"])
				}
				if md["label"] != "Test Artifact Definition" {
					t.Errorf("expected label to be Test Artifact Definition, got %v", md["label"])
				}
				if md["icon"] != "https://example.com/icon.png" {
					t.Errorf("expected icon to be https://example.com/icon.png, got %v", md["icon"])
				}

				// Check UI block
				ui, ok := md["ui"].(map[string]any)
				if !ok {
					t.Fatal("expected ui to be a map")
				}
				if ui["connectionOrientation"] != "environmentDefault" {
					t.Errorf("expected connectionOrientation to be environmentDefault, got %v", ui["connectionOrientation"])
				}
				if ui["environmentDefaultGroup"] != "credentials" {
					t.Errorf("expected environmentDefaultGroup to be credentials, got %v", ui["environmentDefaultGroup"])
				}

				// Check instructions
				instructions, ok := ui["instructions"].([]map[string]any)
				if !ok {
					t.Fatal("expected instructions to be a slice of maps")
				}
				if len(instructions) != 2 {
					t.Errorf("expected 2 instructions, got %d", len(instructions))
				}
				if instructions[0]["label"] != "CLI Setup" {
					t.Errorf("expected first instruction label to be CLI Setup, got %v", instructions[0]["label"])
				}
				// Check that content was read
				content, ok := instructions[0]["content"].(string)
				if !ok || content == "" {
					t.Error("expected instruction content to be a non-empty string")
				}

				// Check exports
				exports, ok := md["export"].([]map[string]any)
				if !ok {
					t.Fatal("expected export to be a slice of maps")
				}
				if len(exports) != 1 {
					t.Errorf("expected 1 export, got %d", len(exports))
				}
				if exports[0]["downloadButtonText"] != "Download Config" {
					t.Errorf("expected downloadButtonText to be Download Config, got %v", exports[0]["downloadButtonText"])
				}
				if exports[0]["fileFormat"] != "yaml" {
					t.Errorf("expected fileFormat to be yaml, got %v", exports[0]["fileFormat"])
				}
				if exports[0]["templateLang"] != "liquid" {
					t.Errorf("expected templateLang to be liquid, got %v", exports[0]["templateLang"])
				}
				// Check that template was read
				template, ok := exports[0]["template"].(string)
				if !ok || template == "" {
					t.Error("expected export template to be a non-empty string")
				}

				// Check schema fields are merged at top level
				if result["$schema"] != "http://json-schema.org/draft-07/schema" {
					t.Errorf("expected $schema at top level, got %v", result["$schema"])
				}
				if result["title"] != "Test Artifact" {
					t.Errorf("expected title to be Test Artifact, got %v", result["title"])
				}
				if result["type"] != "object" {
					t.Errorf("expected type to be object, got %v", result["type"])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := definition.Build(tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Build() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.check != nil {
				tc.check(t, result)
			}
		})
	}
}
