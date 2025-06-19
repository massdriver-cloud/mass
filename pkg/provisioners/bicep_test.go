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
		variables map[string]any
		want      string
	}
	tests := []test{
		{
			name: "same",
			variables: map[string]any{
				"required": []any{"foo", "bar"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
					"bar": map[string]any{
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
			variables: map[string]any{
				"required": []any{"foo", "bar"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
					"bar": map[string]any{
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
			variables: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
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
		want map[string]any
	}
	tests := []test{
		{
			name: "same",
			want: map[string]any{
				"required": []any{"bar", "foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"title": "foo",
						"type":  "string",
					},
					"bar": map[string]any{
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

func TestBicepInitializeStep(t *testing.T) {
	type test struct {
		name         string
		templatePath string
		want         string
	}
	tests := []test{
		{
			name:         "same",
			templatePath: "testdata/bicep/inittest.bicep",
			want: `foobar
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			prov := provisioners.BicepProvisioner{}
			initErr := prov.InitializeStep(testDir, tc.templatePath)
			if initErr != nil {
				t.Fatalf("unexpected error: %s", initErr)
			}

			got, gotErr := os.ReadFile(path.Join(testDir, "template.bicep"))
			if gotErr != nil {
				t.Fatalf("unexpected error: %s", gotErr)
			}

			if string(got) != tc.want {
				t.Errorf("want %v got %v", got, tc.want)
			}
		})
	}
}
