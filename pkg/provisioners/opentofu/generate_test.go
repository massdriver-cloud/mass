package opentofu_test

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners/opentofu"
)

func TestGenerateFiles(t *testing.T) {
	type test struct {
		name   string
		bundle *bundle.Bundle
		want   string
	}
	tests := []test{
		{
			name: "same",
			bundle: &bundle.Bundle{
				Params: map[string]interface{}{
					"required": []interface{}{"param"},
					"properties": map[string]interface{}{
						"param": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Connections: map[string]interface{}{
					"required": []interface{}{"conn"},
					"properties": map[string]interface{}{
						"conn": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			want: ``,
		},
		{
			name: "missingopentofu",
			bundle: &bundle.Bundle{
				Params: map[string]interface{}{
					"required": []interface{}{"param"},
					"properties": map[string]interface{}{
						"param": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Connections: map[string]interface{}{
					"required": []interface{}{"conn"},
					"properties": map[string]interface{}{
						"conn": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			want: `// Auto-generated variable declarations from massdriver.yaml
variable "conn" {
  type = string
}
variable "md_metadata" {
  type = object({
    default_tags = object({
      managed-by  = string
      md-manifest = string
      md-package  = string
      md-project  = string
      md-target   = string
    })
    deployment = object({
      id = string
    })
    name_prefix = string
    observability = object({
      alarm_webhook_url = string
    })
    package = object({
      created_at             = string
      deployment_enqueued_at = string
      previous_status        = string
      updated_at             = string
    })
    target = object({
      contact_email = string
    })
  })
}
variable "param" {
  type = string
}
`,
		},
		{
			name:   "missingmassdriver",
			bundle: &bundle.Bundle{},
			want:   ``,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			err := os.Mkdir(path.Join(testDir, "src"), 0755)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			content, err := os.ReadFile(path.Join("testdata", fmt.Sprintf("%s.tf", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = os.WriteFile(path.Join(testDir, "src", "variables.tf"), content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = opentofu.GenerateFiles(testDir, "src", tc.bundle)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			expectedFilepath := path.Join(testDir, "src", "_massdriver_variables.tf")
			if len(tc.want) > 0 {
				got, err := os.ReadFile(expectedFilepath)
				if err != nil {
					t.Fatalf("%d, unexpected error", err)
				}

				if string(got) != tc.want {
					t.Errorf("got %s want %s", got, tc.want)
				}
			} else {
				if _, err := os.Stat(expectedFilepath); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("file exists when it shouldn't")
				}
			}
		})
	}
}
