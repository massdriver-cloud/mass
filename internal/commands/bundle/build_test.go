package bundle_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mdbundle "github.com/massdriver-cloud/mass/internal/bundle"
	cmdbundle "github.com/massdriver-cloud/mass/internal/commands/bundle"
)

func TestValidateSchema(t *testing.T) {
	schema := `{"type":"object","required":["name","params"],"properties":{"name":{"type":"string"},"params":{"type":"object"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json-schemas/bundle.json" {
			_, _ = w.Write([]byte(schema))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	params := map[string]any{"properties": map[string]any{}}

	t.Run("valid bundle passes", func(t *testing.T) {
		b := &mdbundle.Bundle{Name: "example", Params: params}
		if err := cmdbundle.ValidateSchema(b, server.URL); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("schema-invalid bundle is rejected", func(t *testing.T) {
		b := &mdbundle.Bundle{Params: params} // missing required name
		err := cmdbundle.ValidateSchema(b, server.URL)
		if err == nil || !strings.Contains(err.Error(), "schema validation") {
			t.Fatalf("want a schema validation error, got: %v", err)
		}
	})

	t.Run("unreachable schema is rejected", func(t *testing.T) {
		b := &mdbundle.Bundle{Name: "example", Params: params}
		if err := cmdbundle.ValidateSchema(b, "http://127.0.0.1:0"); err == nil {
			t.Fatal("want an error when the schema cannot be fetched")
		}
	})
}
