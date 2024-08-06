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
		mdParamsSchema map[string]interface{}
		airlockSchema  *schema.Schema
		want           map[string]interface{}
	}
	tests := []test{
		{
			name: "none",
			mdParamsSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
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
			want: map[string]interface{}{
				"properties": map[string]interface{}{},
				"required":   []interface{}{},
			},
		},
		{
			name: "missing airlock params",
			mdParamsSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"const": "bar",
					},
				},
				"required": []any{"foo"},
			},
			airlockSchema: &schema.Schema{},
			want: map[string]interface{}{
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"const": "bar",
					},
				},
				"required": []any{"foo"},
			},
		},
		{
			name:           "missing md params",
			mdParamsSchema: map[string]interface{}{},
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
			want: map[string]interface{}{
				"properties": map[string]interface{}{},
				"required":   []interface{}{},
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
