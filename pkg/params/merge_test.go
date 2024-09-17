package params_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/params"
)

func TestMergeSchemas(t *testing.T) {
	type test struct {
		name string
		m1   map[string]any
		m2   map[string]any
		want map[string]any
	}
	tests := []test{
		{
			name: "basic",
			m1: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
				},
			},
			m2: map[string]any{
				"required": []any{"bar"},
				"properties": map[string]any{
					"bar": map[string]any{
						"type": "string",
					},
				},
			},
			want: map[string]any{
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
		},
		{
			name: "m1 empty",
			m1:   map[string]any{},
			m2: map[string]any{
				"required": []any{"bar"},
				"properties": map[string]any{
					"bar": map[string]any{
						"type": "string",
					},
				},
			},
			want: map[string]any{
				"required": []any{"bar"},
				"properties": map[string]any{
					"bar": map[string]any{
						"type": "string",
					},
				},
			},
		},
		{
			name: "m2 empty",
			m1: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
				},
			},
			m2: map[string]any{},
			want: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
				},
			},
		},
		{
			name: "collision",
			m1: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "string",
					},
				},
			},
			m2: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "bool",
					},
				},
			},
			want: map[string]any{
				"required": []any{"foo"},
				"properties": map[string]any{
					"foo": map[string]any{
						"type": "bool",
					},
				},
			},
		},
		{
			name: "both empty",
			m1:   map[string]any{},
			m2:   map[string]any{},
			want: map[string]any{
				"required":   []any{},
				"properties": map[string]any{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := params.MergeSchemas(tc.m1, tc.m2)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}
