package jsonschema_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
)

func TestValidate(t *testing.T) {
	type test struct {
		name         string
		schemaPath   string
		documentPath string
		want         bool
	}
	tests := []test{
		{
			name:         "ValidDocument",
			schemaPath:   "testdata/valid-schema.json",
			documentPath: "testdata/valid-document.json",
			want:         true,
		},
		{
			name:         "InvalidDocument",
			schemaPath:   "testdata/valid-schema.json",
			documentPath: "testdata/invalid-document.json",
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := jsonschema.Validate(tc.schemaPath, tc.documentPath)
			if err != nil {
				t.Errorf("Error during validation: %s", err)
			}

			if got.Valid() != tc.want {
				t.Errorf("got %t want %t", got.Valid(), tc.want)
			}
		})
	}
}
