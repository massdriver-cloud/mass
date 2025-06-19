package provisioners_test

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

func TestHelmReadProvisionerInputs(t *testing.T) {
	type test struct {
		name string
		want map[string]any
	}
	tests := []test{
		{
			name: "same",
			want: map[string]any{
				"required": []any{"foo", "baz"},
				"properties": map[string]any{
					"foo": map[string]any{
						"title":   "foo",
						"type":    "string",
						"default": "bar",
					},
					"baz": map[string]any{
						"title":   "baz",
						"type":    "string",
						"default": "qux",
					},
				},
				"type": "object",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			content, err := os.ReadFile(path.Join("testdata", "helm", fmt.Sprintf("%s.yaml", tc.name)))
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			testFile := path.Join(testDir, "values.yaml")
			err = os.WriteFile(testFile, content, 0644)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			prov := provisioners.HelmProvisioner{}
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

func TestHelmInitializeStep(t *testing.T) {
	type test struct {
		name      string
		chartPath string
	}
	tests := []test{
		{
			name:      "same",
			chartPath: "testdata/helm/initializetest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			prov := provisioners.HelmProvisioner{}
			initErr := prov.InitializeStep(testDir, tc.chartPath)
			if initErr != nil {
				t.Fatalf("unexpected error: %s", initErr)
			}

			want := []string{}
			wanttErr := filepath.Walk(tc.chartPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if tc.chartPath == path {
					return nil
				}
				want = append(want, info.Name())
				return nil
			})
			if wanttErr != nil {
				t.Fatalf("unexpected error: %s", wanttErr)
			}

			got := []string{}
			gotErr := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if testDir == path {
					return nil
				}
				got = append(got, info.Name())
				return nil
			})
			if gotErr != nil {
				t.Fatalf("unexpected error: %s", gotErr)
			}

			if len(got) != len(want) {
				t.Errorf("want %v got %v", got, want)
			}
			for _, curr := range got {
				if !slices.Contains(want, curr) {
					t.Errorf("%v doesn't exist in %v", curr, want)
				}
			}
		})
	}
}
