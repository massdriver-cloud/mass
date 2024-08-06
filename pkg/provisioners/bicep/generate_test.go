package bicep_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners/bicep"
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
			want: `param param string
param conn string
param md_metadata object
`,
		},
		{
			name: "missingbicep",
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
			want: `param foo string

// Auto-generated param declarations from massdriver.yaml
param conn string
param md_metadata object
param param string
`,
		},
		{
			name:   "missingmassdriver",
			bundle: &bundle.Bundle{},
			want: `param foo string
param md_metadata object
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			err := os.Mkdir(path.Join(testDir, "src"), 0755)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			content, err := os.ReadFile(path.Join("testdata", fmt.Sprintf("%s.bicep", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = os.WriteFile(path.Join(testDir, "src", "template.bicep"), content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = bicep.GenerateFiles(testDir, "src", tc.bundle)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			got, err := os.ReadFile(path.Join(testDir, "src", "template.bicep"))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if string(got) != tc.want {
				t.Errorf("got %s want %s", got, tc.want)
			}
		})
	}
}
