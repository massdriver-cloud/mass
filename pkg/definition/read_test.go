package definition_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRead(t *testing.T) {
	want := map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema",
		"$md": map[string]any{
			"name": "foo",
		},
		"type":  "object",
		"title": "Test Artifact",
		"properties": map[string]any{
			"data": map[string]any{
				"type": "object",
			},
			"specs": map[string]any{
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
			file: filepath.Join("testdata", "simple-artifact.json"),
		},
		{
			name: "yaml",
			file: filepath.Join("testdata", "simple-artifact.yaml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mdClient := client.Client{
				GQL: gqlmock.NewClientWithSingleJSONResponse(map[string]any{"data": map[string]any{}}),
			}

			got, err := definition.Read(context.Background(), &mdClient, tc.file)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}
