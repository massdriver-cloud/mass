package provisioners_test

import (
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func TestOpentofuExportMassdriverInputs(t *testing.T) {
	type test struct {
		name      string
		variables map[string]interface{}
		want      string
	}
	tests := []test{
		{
			name: "same",
			variables: map[string]interface{}{
				"required": []interface{}{"foo", "bar"},
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type": "string",
					},
					"bar": map[string]interface{}{
						"type": "string",
					},
				},
			},
			want: ``,
		},
		{
			name: "missingopentofu",
			variables: map[string]interface{}{
				"required": []interface{}{"foo", "bar"},
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type": "string",
					},
					"bar": map[string]interface{}{
						"type": "string",
					},
				},
			},
			want: `// Auto-generated variable declarations from massdriver.yaml
variable "bar" {
  type = string
}
`,
		},
		{
			name: "missingmassdriver",
			variables: map[string]interface{}{
				"required": []interface{}{"foo"},
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type": "string",
					},
				},
			},
			want: ``,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			content, err := os.ReadFile(path.Join("testdata", "opentofu", fmt.Sprintf("%s.tf", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = os.WriteFile(path.Join(testDir, "variables.tf"), content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			prov := provisioners.OpentofuProvisioner{}
			err = prov.ExportMassdriverInputs(testDir, tc.variables)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			expectedFilepath := path.Join(testDir, "_massdriver_variables.tf")
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

func TestOpentofuReadProvisionerInputs(t *testing.T) {
	type test struct {
		name string
		want map[string]interface{}
	}
	tests := []test{
		{
			name: "same",
			want: map[string]interface{}{
				"required": []interface{}{"foo", "bar"},
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"title": "foo",
						"type":  "string",
					},
					"bar": map[string]interface{}{
						"title": "bar",
						"type":  "string",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			content, err := os.ReadFile(path.Join("testdata", "opentofu", fmt.Sprintf("%s.tf", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			err = os.WriteFile(path.Join(testDir, "variables.tf"), content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			prov := provisioners.OpentofuProvisioner{}
			got, err := prov.ReadProvisionerInputs(testDir)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("want %v got %v", got, tc.want)
			}
		})
	}
}