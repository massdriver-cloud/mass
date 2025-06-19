package bundle_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

var mdMetadataMap = map[string]any{
	"properties": map[string]any{
		"default_tags": map[string]any{
			"properties": map[string]any{
				"managed-by":  map[string]any{"type": "string"},
				"md-manifest": map[string]any{"type": "string"},
				"md-package":  map[string]any{"type": "string"},
				"md-project":  map[string]any{"type": "string"},
				"md-target":   map[string]any{"type": "string"},
			},
			"required": []any{"managed-by", "md-manifest", "md-package", "md-project", "md-target"},
			"type":     "object",
		},
		"deployment": map[string]any{
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
			"required": []any{"id"},
			"type":     "object",
		},
		"name_prefix": map[string]any{"type": "string"},
		"observability": map[string]any{
			"properties": map[string]any{
				"alarm_webhook_url": map[string]any{"type": "string"},
			},
			"required": []any{"alarm_webhook_url"},
			"type":     "object",
		},
		"package": map[string]any{
			"properties": map[string]any{
				"created_at":             map[string]any{"type": "string"},
				"deployment_enqueued_at": map[string]any{"type": "string"},
				"previous_status":        map[string]any{"type": "string"},
				"updated_at":             map[string]any{"type": "string"},
			},
			"required": []any{"created_at", "deployment_enqueued_at", "previous_status", "updated_at"},
			"type":     "object",
		},
		"target": map[string]any{
			"properties": map[string]any{
				"contact_email": map[string]any{"type": "string"},
			},
			"required": []any{"contact_email"},
			"type":     "object"},
	},
	"required": []any{"default_tags", "deployment", "name_prefix", "observability", "package", "target"},
	"type":     "object",
}

func TestCombineParamsConnsMetadata(t *testing.T) {
	type test struct {
		name   string
		bundle *bundle.Bundle
		want   map[string]any
	}
	tests := []test{
		{
			name: "none",
			bundle: &bundle.Bundle{
				Params: map[string]any{
					"required": []any{"param"},
					"properties": map[string]any{
						"param": map[string]any{
							"type": "string",
						},
					},
				},
				Connections: map[string]any{
					"required": []any{"conn"},
					"properties": map[string]any{
						"conn": map[string]any{
							"type": "string",
						},
					},
				},
			},
			want: map[string]any{
				"required": []any{"param", "conn", "md_metadata"},
				"properties": map[string]any{
					"param": map[string]any{
						"type": "string",
					},
					"conn": map[string]any{
						"type": "string",
					},
					"md_metadata": mdMetadataMap,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.bundle.CombineParamsConnsMetadata()

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}
