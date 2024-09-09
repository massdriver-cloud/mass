package bundle

import (
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

type DereferenceTarget struct {
	schema   map[string]interface{}
	label    string
	callback *Schema
}

func (b *Bundle) DereferenceSchemas(path string, c *restclient.MassdriverClient) error {
	cwd := filepath.Dir(path)
	tasks := []DereferenceTarget{
		{schema: b.Artifacts.ToMap(), label: "artifacts", callback: b.Artifacts},
		{schema: b.Params.ToMap(), label: "params", callback: b.Params},
		{schema: b.Connections.ToMap(), label: "connections", callback: b.Connections},
		{schema: b.UI, label: "ui", callback: nil},
	}

	for _, task := range tasks {
		if task.schema == nil {
			task.schema = map[string]interface{}{
				"properties": make(map[string]interface{}),
			}
		}

		dereferencedSchemaInterface, err := jsonschema.Dereference(task.schema, jsonschema.DereferenceOptions{Client: c, Cwd: cwd})
		if err != nil {
			return err
		}

		dereferencedSchema, ok := dereferencedSchemaInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("hydrated %s is not a map", task.label)
		}

		if task.callback != nil {
			task.callback.FromMap(dereferencedSchema)

		} else {
			task.schema = dereferencedSchema
		}
	}

	return nil
}
