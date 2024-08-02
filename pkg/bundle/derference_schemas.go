package bundle

import (
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

type DereferenceTarget struct {
	schema *map[string]*Schema
	label  string
}

func (b *Bundle) DereferenceSchemas(path string, c *restclient.MassdriverClient) error {
	cwd := filepath.Dir(path)
	tasks := []DereferenceTarget{
		{schema: &b.Artifacts, label: "artifacts"},
		{schema: &b.Params, label: "params"},
		{schema: &b.Connections, label: "connections"},
		{schema: &b.UI, label: "ui"},
	}

	for _, task := range tasks {
		if task.schema == nil {
			*task.schema = map[string]interface{}{
				"properties": make(map[string]interface{}),
			}
		}

		dereferencedSchema, err := jsonschema.Dereference(*task.schema, jsonschema.DereferenceOptions{Client: c, Cwd: cwd})

		if err != nil {
			return err
		}

		var ok bool
		*task.schema, ok = dereferencedSchema.(map[string]interface{})

		if !ok {
			return fmt.Errorf("hydrated %s is not a map", task.label)
		}
	}

	return nil
}
