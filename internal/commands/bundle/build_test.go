package bundle_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mdbundle "github.com/massdriver-cloud/mass/internal/bundle"
	cmdbundle "github.com/massdriver-cloud/mass/internal/commands/bundle"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
)

type fakePublishedBundleAPI struct {
	version       string
	cardinalities map[string]string
	err           error
}

func (f fakePublishedBundleAPI) PublishedCardinalities(_ context.Context, _ string) (string, map[string]string, error) {
	return f.version, f.cardinalities, f.err
}

func TestWarnCardinalityChanges(t *testing.T) {
	tests := []struct {
		name      string
		published fakePublishedBundleAPI
		deps      map[string]mdbundle.Dependency
		want      []string
	}{
		{
			name:      "one to many warns",
			published: fakePublishedBundleAPI{version: "1.2.0", cardinalities: map[string]string{"queues": "one"}},
			deps:      map[string]mdbundle.Dependency{"queues": {Cardinality: "many"}},
			want: []string{
				`dependency "queues" changes cardinality from one to many since aws-worker@1.2.0`,
				"The variable becomes a map; the existing line becomes a set of one.",
			},
		},
		{
			name:      "many to one warns",
			published: fakePublishedBundleAPI{version: "1.2.0", cardinalities: map[string]string{"queues": "many"}},
			deps:      map[string]mdbundle.Dependency{"queues": {}},
			want: []string{
				`dependency "queues" changes cardinality from many to one since aws-worker@1.2.0`,
				"The variable becomes an object; on upgrade the field's connections are removed and the user connects one source.",
			},
		},
		{
			name:      "same cardinality does not warn",
			published: fakePublishedBundleAPI{version: "1.2.0", cardinalities: map[string]string{"network": "one", "queues": "many"}},
			deps: map[string]mdbundle.Dependency{
				"network": {},
				"queues":  {Cardinality: "many"},
			},
		},
		{
			name:      "new field does not warn",
			published: fakePublishedBundleAPI{version: "1.2.0", cardinalities: map[string]string{"network": "one"}},
			deps:      map[string]mdbundle.Dependency{"queues": {Cardinality: "many"}},
		},
		{
			name:      "no published version does not warn",
			published: fakePublishedBundleAPI{err: fmt.Errorf("get bundle aws-worker: %w", gql.ErrNotFound)},
			deps:      map[string]mdbundle.Dependency{"queues": {Cardinality: "many"}},
		},
		{
			name:      "API failure does not warn",
			published: fakePublishedBundleAPI{err: errors.New("dial tcp: connection refused")},
			deps:      map[string]mdbundle.Dependency{"queues": {Cardinality: "many"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &mdbundle.Bundle{Name: "aws-worker", Dependencies: tc.deps}
			var out bytes.Buffer
			cmdbundle.WarnCardinalityChanges(t.Context(), tc.published, b, &out)

			got := out.String()
			if len(tc.want) == 0 && got != "" {
				t.Fatalf("want no output, got %q", got)
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("output %q missing %q", got, want)
				}
			}
		})
	}
}

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
