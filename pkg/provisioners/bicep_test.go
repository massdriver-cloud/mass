package provisioners_test

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func TestBicepExportMassdriverInputs(t *testing.T) {
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
			want: `param foo string
param bar string
`,
		},
		{
			name: "missingbicep",
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
			want: `param foo string

// Auto-generated param declarations from massdriver.yaml
param bar string
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
			want: `param foo string
param bar string
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			content, err := os.ReadFile(path.Join("testdata", "bicep", fmt.Sprintf("%s.bicep", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			testFile := path.Join(testDir, "template.bicep")
			err = os.WriteFile(testFile, content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			prov := provisioners.BicepProvisioner{}
			err = prov.ExportMassdriverInputs(testDir, tc.variables)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			got, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if string(got) != tc.want {
				t.Errorf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestBicepReadProvisionerInputs(t *testing.T) {
	type test struct {
		name string
		want map[string]interface{}
	}
	tests := []test{
		{
			name: "same",
			want: map[string]interface{}{
				"required": []interface{}{"bar", "foo"},
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
				"type": "object",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			content, err := os.ReadFile(path.Join("testdata", "bicep", fmt.Sprintf("%s.bicep", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			testFile := path.Join(testDir, "template.bicep")
			err = os.WriteFile(testFile, content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			prov := provisioners.BicepProvisioner{}
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
