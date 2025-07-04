package jsonschema_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
)

func TestValidateFile(t *testing.T) {
	tests := []struct {
		name         string
		schemaPath   string
		documentPath string
		wantErr      bool
	}{
		{
			name:         "valid document",
			schemaPath:   "testdata/valid-schema.json",
			documentPath: "testdata/valid-document.json",
			wantErr:      false,
		},
		{
			name:         "invalid document",
			schemaPath:   "testdata/valid-schema.json",
			documentPath: "testdata/invalid-document.json",
			wantErr:      true,
		},
		{
			name:         "nonexistent document",
			schemaPath:   "testdata/valid-schema.json",
			documentPath: "testdata/nonexistent.json",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromFile(tt.schemaPath)
			if err != nil {
				t.Fatalf("Failed to load schema: %v", err)
			}

			err = jsonschema.ValidateFile(schema, tt.documentPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBytes(t *testing.T) {
	validDocumentBytes := []byte(`{
		"checked": false,
		"dimensions": {"width": 5, "height": 10},
		"id": 1,
		"name": "A green door",
		"price": 12.5,
		"tags": ["home", "green"]
	}`)

	invalidDocumentBytes := []byte(`{
		"checked": "should be boolean"
	}`)

	malformedJSON := []byte(`{"invalid": json}`)

	tests := []struct {
		name          string
		schemaPath    string
		documentBytes []byte
		wantErr       bool
	}{
		{
			name:          "valid document bytes",
			schemaPath:    "testdata/valid-schema.json",
			documentBytes: validDocumentBytes,
			wantErr:       false,
		},
		{
			name:          "invalid document bytes",
			schemaPath:    "testdata/valid-schema.json",
			documentBytes: invalidDocumentBytes,
			wantErr:       true,
		},
		{
			name:          "malformed JSON bytes",
			schemaPath:    "testdata/valid-schema.json",
			documentBytes: malformedJSON,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromFile(tt.schemaPath)
			if err != nil {
				t.Fatalf("Failed to load schema: %v", err)
			}

			err = jsonschema.ValidateBytes(schema, tt.documentBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateGo(t *testing.T) {
	validDocument := map[string]any{
		"checked":    false,
		"dimensions": map[string]any{"width": 5, "height": 10},
		"id":         1,
		"name":       "A green door",
		"price":      12.5,
		"tags":       []string{"home", "green"},
	}

	invalidDocument := map[string]any{
		"checked": "should be boolean",
	}

	validStruct := struct {
		Checked    bool           `json:"checked"`
		Dimensions map[string]int `json:"dimensions"`
		ID         int            `json:"id"`
		Name       string         `json:"name"`
		Price      float64        `json:"price"`
		Tags       []string       `json:"tags"`
	}{
		Checked:    false,
		Dimensions: map[string]int{"width": 5, "height": 10},
		ID:         1,
		Name:       "A green door",
		Price:      12.5,
		Tags:       []string{"home", "green"},
	}

	tests := []struct {
		name       string
		schemaPath string
		document   any
		wantErr    bool
	}{
		{
			name:       "valid Go map",
			schemaPath: "testdata/valid-schema.json",
			document:   validDocument,
			wantErr:    false,
		},
		{
			name:       "invalid Go map",
			schemaPath: "testdata/valid-schema.json",
			document:   invalidDocument,
			wantErr:    true,
		},
		{
			name:       "valid Go struct",
			schemaPath: "testdata/valid-schema.json",
			document:   validStruct,
			wantErr:    false,
		},
		{
			name:       "unmarshalable Go object",
			schemaPath: "testdata/valid-schema.json",
			document:   make(chan int),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromFile(tt.schemaPath)
			if err != nil {
				t.Fatalf("Failed to load schema: %v", err)
			}

			err = jsonschema.ValidateGo(schema, tt.document)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
