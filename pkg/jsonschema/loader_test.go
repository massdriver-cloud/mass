package jsonschema_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
)

func TestLoadSchemaFromFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	tests := []struct {
		name    string
		path    string
		wantErr bool
		wantID  string
	}{
		{
			name:    "valid schema with relative path",
			path:    "testdata/schema.json",
			wantErr: false,
		},
		{
			name:    "valid schema with absolute path",
			path:    filepath.Join(pwd, "testdata/schema.json"),
			wantErr: false,
		},
		{
			name:    "valid schema with file prefix",
			path:    "file://./testdata/schema.json",
			wantErr: false,
		},
		{
			name:    "another valid schema",
			path:    "testdata/valid-schema.json",
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			path:    "testdata/nonexistent.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchemaFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && schema == nil {
				t.Errorf("LoadSchemaFromFile() returned nil schema")
			}
		})
	}
}

func TestLoadSchemaFromURL(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid-schema.json":
			http.ServeFile(w, r, "testdata/valid-schema.json")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid URL",
			url:     server.URL + "/valid-schema.json",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     server.URL + "/nonexistent.json",
			wantErr: true,
		},
		{
			name:    "malformed URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchemaFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && schema == nil {
				t.Errorf("LoadSchemaFromURL() returned nil schema")
			}
		})
	}
}

func TestLoadSchemaFromGo(t *testing.T) {
	tests := []struct {
		name    string
		obj     any
		wantErr bool
	}{
		{
			name: "valid Go object",
			obj: map[string]any{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type":    "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "struct object",
			obj: struct {
				Schema string `json:"$schema"`
				Type   string `json:"type"`
			}{
				Schema: "http://json-schema.org/draft-07/schema#",
				Type:   "object",
			},
			wantErr: false,
		},
		{
			name:    "unmarshalable object",
			obj:     make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := jsonschema.LoadSchemaFromGo(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchemaFromGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && schema == nil {
				t.Errorf("LoadSchemaFromGo() returned nil schema")
			}
		})
	}
}

func TestLoadSchemaFromReader(t *testing.T) {
	validSchemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	invalidJSON := `{"invalid": json}`

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid JSON schema",
			content: validSchemaJSON,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			content: invalidJSON,
			wantErr: true,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tt.content))
			schema, err := jsonschema.LoadSchemaFromReader(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchemaFromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && schema == nil {
				t.Errorf("LoadSchemaFromReader() returned nil schema")
			}
		})
	}
}
