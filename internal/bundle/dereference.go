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
	b.hydrateDependencySchema()

	// stripID drops "$id" from the params schema; dependencies keep it.
	tasks := []struct {
		schema  *map[string]any
		label   string
		stripID bool
	}{
		{schema: &b.Params, label: "params", stripID: true},
		{schema: &b.dependencySchema, label: "dependencies", stripID: false},
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
