package resourcetype_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRead(t *testing.T) {
	want := map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema",
		"$md": map[string]any{
			"name": "foo",
		},
		"type":  "object",
		"title": "Test Resource Type",
		"properties": map[string]any{
			"foo": map[string]any{
				"type": "object",
			},
			"bar": map[string]any{
				"type": "object",
			},
		},
	}
	type test struct {
		name string
		file string
	}
	tests := []test{
		{
			name: "json",
			file: filepath.Join("testdata", "simple-resource.json"),
		},
		{
			name: "yaml",
			file: filepath.Join("testdata", "simple-resource.yaml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mdClient := client.Client{
				GQL: gqlmock.NewClientWithSingleJSONResponse(map[string]any{"data": map[string]any{}}),
			}

			got, err := resourcetype.Read(t.Context(), &mdClient, tc.file)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}
