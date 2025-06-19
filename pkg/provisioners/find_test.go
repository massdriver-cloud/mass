package provisioners_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/airlock/pkg/schema"
	"github.com/massdriver-cloud/mass/pkg/provisioners"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestFindMissingFromAirlock(t *testing.T) {
	type test struct {
		name           string
		mdParamsSchema map[string]any
		airlockSchema  *schema.Schema
		want           map[string]any
	}
	tests := []test{
		{
			name: "none",
			mdParamsSchema: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{
						"const": "bar",
					},
				},
				"required": []any{"foo"},
			},
			airlockSchema: &schema.Schema{
				Properties: orderedmap.New[string, *schema.Schema](orderedmap.WithInitialData[string, *schema.Schema](
					orderedmap.Pair[string, *schema.Schema]{
						Key: "foo",
						Value: &schema.Schema{
							Const: "bar",
						},
					},
				)),
				Required: []string{"foo"},
			},
			want: map[string]any{
				"properties": map[string]any{},
				"required":   []any{},
			},
		},
		{
			name: "missing airlock params",
			mdParamsSchema: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{
						"const": "bar",
					},
				},
				"required": []any{"foo"},
			},
			airlockSchema: &schema.Schema{},
			want: map[string]any{
				"properties": map[string]any{
					"foo": map[string]any{
						"const": "bar",
					},
				},
				"required": []any{"foo"},
			},
		},
		{
			name:           "missing md params",
			mdParamsSchema: map[string]any{},
			airlockSchema: &schema.Schema{
				Properties: orderedmap.New[string, *schema.Schema](orderedmap.WithInitialData[string, *schema.Schema](
					orderedmap.Pair[string, *schema.Schema]{
						Key: "foo",
						Value: &schema.Schema{
							Const: "bar",
						},
					},
				)),
				Required: []string{"foo"},
			},
			want: map[string]any{
				"properties": map[string]any{},
				"required":   []any{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := provisioners.FindMissingFromAirlock(tc.mdParamsSchema, tc.airlockSchema)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}
