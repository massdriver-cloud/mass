package bundle

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/resourcetype"
)

// SchemaResolver fetches a published resource-type schema by name (returned as
// a generic map). The bundle dereferencer invokes it for every massdriver $ref
// encountered while walking the bundle's schemas.
type SchemaResolver func(ctx context.Context, name string) (map[string]any, error)

// DereferenceSchemas resolves all $ref entries in the bundle's schemas. Massdriver
// $refs are looked up via the supplied resolver — pass
// [resourcetype.NewMassdriverResolver] in production, or a hand-rolled stub in
// tests.
func (b *Bundle) DereferenceSchemas(path string, resolver SchemaResolver) error {
	cwd := filepath.Dir(path)

	// The stripID is a hack to get around the issue of the UI choking if the params schema has 2 or more of the same $id in it.
	// We need the "$id" in artifacts and connections, but we need to strip it out of params and ui schemas, hence the conditional.
	// This logic should be removed when we have a better solution for this in the UI/API - probably after resource types are in OCI
	tasks := []struct {
		schema  *map[string]any
		label   string
		stripID bool
	}{
		{schema: &b.Artifacts, label: "artifacts", stripID: false},
		{schema: &b.Params, label: "params", stripID: true},
		{schema: &b.Connections, label: "connections", stripID: false},
		{schema: &b.UI, label: "ui", stripID: true},
	}

	for _, task := range tasks {
		if task.schema == nil {
			*task.schema = map[string]any{
				"properties": make(map[string]any),
			}
		}

		dereferencedSchema, err := resourcetype.DereferenceSchema(*task.schema, resourcetype.DereferenceOptions{Resolver: resolver, Cwd: cwd, StripID: task.stripID})

		if err != nil {
			return err
		}

		var ok bool
		*task.schema, ok = dereferencedSchema.(map[string]any)

		if !ok {
			return fmt.Errorf("hydrated %s is not a map", task.label)
		}
	}

	return nil
}
