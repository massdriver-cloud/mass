package provisioners_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
)

var md_metadata_map = map[string]interface{}{
	"properties": map[string]interface{}{
		"default_tags": map[string]interface{}{
			"properties": map[string]interface{}{
				"managed-by":  map[string]interface{}{"type": "string"},
				"md-manifest": map[string]interface{}{"type": "string"},
				"md-package":  map[string]interface{}{"type": "string"},
				"md-project":  map[string]interface{}{"type": "string"},
				"md-target":   map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"managed-by", "md-manifest", "md-package", "md-project", "md-target"},
			"type":     "object",
		},
		"deployment": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"id"},
			"type":     "object",
		},
		"name_prefix": map[string]interface{}{"type": "string"},
		"observability": map[string]interface{}{
			"properties": map[string]interface{}{
				"alarm_webhook_url": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"alarm_webhook_url"},
			"type":     "object",
		},
		"package": map[string]interface{}{
			"properties": map[string]interface{}{
				"created_at":             map[string]interface{}{"type": "string"},
				"deployment_enqueued_at": map[string]interface{}{"type": "string"},
				"previous_status":        map[string]interface{}{"type": "string"},
				"updated_at":             map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"created_at", "deployment_enqueued_at", "previous_status", "updated_at"},
			"type":     "object",
		},
		"target": map[string]interface{}{
			"properties": map[string]interface{}{
				"contact_email": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"contact_email"},
			"type":     "object"},
	},
	"required": []interface{}{"default_tags", "deployment", "name_prefix", "observability", "package", "target"},
	"type":     "object",
}

func TestCombineParamsConnsMetadata(t *testing.T) {
	type test struct {
		name   string
		bundle *bundle.Bundle
		want   map[string]interface{}
	}
	tests := []test{
		{
			name: "none",
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
			want: map[string]interface{}{
				"required": []interface{}{"param", "conn", "md_metadata"},
				"properties": map[string]interface{}{
					"param": map[string]interface{}{
						"type": "string",
					},
					"conn": map[string]interface{}{
						"type": "string",
					},
					"md_metadata": md_metadata_map,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := provisioners.CombineParamsConnsMetadata(tc.bundle)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}
