package imprt_test

import (
	"os"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/bundle/imprt"
)

func TestImportParams(t *testing.T) {
	type test struct {
		name       string
		mdyamlPath string
		tfContent  string
		want       string
	}
	tests := []test{
		{
			name:       "empty-params",
			mdyamlPath: "testdata/empty-massdriver.yaml",
			tfContent:  `variable "new" {type = string}`,
			want: `schema: draft-07
name: "test-bundle"
description: "Bundles to test things"
source_url: github.com/YOUR_NAME_HERE/test-bundle
access: private
type: infrastructure
steps:
    - path: src
      provisioner: opentofu
params:
    properties:
        new:
            title: new
            type: string
    required:
        - new
connections: {}
artifacts: {}
ui: {}
`,
		},
		{
			name:       "same-params",
			mdyamlPath: "testdata/foo-massdriver.yaml",
			tfContent:  `variable "foo" {type = string}`,
			want: `schema: draft-07
name: "test-bundle"
description: "Bundles to test things"
source_url: github.com/YOUR_NAME_HERE/test-bundle
access: private
type: infrastructure
steps:
  - path: src
    provisioner: opentofu
params:
  properties:
    foo:
      type: string
  required:
    - foo
connections: {}
artifacts: {}
ui: {}
`,
		},
		{
			name:       "simple-add",
			mdyamlPath: "testdata/foo-massdriver.yaml",
			tfContent:  `variable "new" {type = string}`,
			want: `schema: draft-07
name: "test-bundle"
description: "Bundles to test things"
source_url: github.com/YOUR_NAME_HERE/test-bundle
access: private
type: infrastructure
steps:
    - path: src
      provisioner: opentofu
params:
    properties:
        foo:
            type: string
        new:
            title: new
            type: string
    required:
        - foo
        - new
connections: {}
artifacts: {}
ui: {}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup temporary directory for testing
			testDir := t.TempDir()

			// Copy the massdriver.yaml file to the temporary directory
			mdYamlContent, err := os.ReadFile(tc.mdyamlPath)
			if err != nil {
				t.Fatalf("Failed to read massdriver.yaml file: %v", err)
			}
			err = os.WriteFile(path.Join(testDir, "massdriver.yaml"), mdYamlContent, 0644)
			if err != nil {
				t.Fatalf("Failed to write massdriver.yaml file: %v", err)
			}

			err = os.MkdirAll(path.Join(testDir, "src"), 0755)
			if err != nil {
				t.Fatalf("Failed to create src directory: %v", err)
			}
			err = os.WriteFile(path.Join(testDir, "src", "main.tf"), []byte(tc.tfContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write main.tf file: %v", err)
			}

			// Run the ImportParams function
			err = imprt.Run(testDir, true)
			if err != nil {
				t.Fatalf("ImportParams returned an error: %v", err)
			}

			// Read the updated massdriver.yaml file
			got, err := os.ReadFile(path.Join(testDir, "massdriver.yaml"))
			if err != nil {
				t.Fatalf("Failed to read updated massdriver.yaml: %v", err)
			}

			// Verify the updated content
			if string(got) != tc.want {
				t.Errorf("Updated massdriver.yaml content does not match expected content.\nWant:\n%s\nGot:\n%s", tc.want, string(got))
			}
		})
	}
}
