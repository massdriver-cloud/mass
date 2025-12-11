package bundle

import (
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/definition"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func (b *Bundle) DereferenceSchemas(path string, mdClient *client.Client) error {
	cwd := filepath.Dir(path)

	tasks := []struct {
		schema *map[string]any
		label  string
	}{
		{schema: &b.Artifacts, label: "artifacts"},
		{schema: &b.Params, label: "params"},
		{schema: &b.Connections, label: "connections"},
		{schema: &b.UI, label: "ui"},
	}

	for _, task := range tasks {
		if task.schema == nil {
			*task.schema = map[string]any{
				"properties": make(map[string]any),
			}
		}

		dereferencedSchema, err := definition.DereferenceSchema(*task.schema, definition.DereferenceOptions{Client: mdClient, Cwd: cwd})

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
