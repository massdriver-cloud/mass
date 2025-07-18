package bundle_test

import (
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

func TestEnsureBooleansHaveDefault(t *testing.T) {

	testCases := []struct {
		name  string
		input map[string]any
		want  map[string]any
	}{
		{
			name:  "sets default to false for boolean without default",
			input: map[string]any{"type": "boolean"},
			want:  map[string]any{"type": "boolean", "default": false},
		},
		{
			name: "sets default in nested object",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nested": map[string]any{"type": "boolean"},
				},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nested": map[string]any{"type": "boolean", "default": false},
				},
			},
		},
		{
			name:  "does not change boolean with default",
			input: map[string]any{"type": "boolean", "default": true},
			want:  map[string]any{"type": "boolean", "default": true},
		},
		{
			name:  "does not change non-boolean type",
			input: map[string]any{"type": "string"},
			want:  map[string]any{"type": "string"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := bundle.ApplyTransformations(tc.input, []func(map[string]interface{}) error{bundle.EnsureBooleansHaveDefault})
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if fmt.Sprint(tc.input) != fmt.Sprint(tc.want) {
				t.Errorf("got %v, want %v", tc.input, tc.want)
			}
		})
	}
}
