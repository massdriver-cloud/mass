package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

const idURLPattern = "https://schemas.massdriver.cloud/schemas/bundles/%s/schema-%s.json"
const jsonSchemaURLPattern = "http://json-schema.org/%s/schema"

func (b *Bundle) WriteSchemas(buildPath string) error {
	type WriteTask struct {
		schema interface{}
		label  string
	}

	mkdirErr := os.MkdirAll(buildPath, 0755)

	if mkdirErr != nil {
		return mkdirErr
	}

	tasks := []WriteTask{
		{schema: b.Artifacts, label: "artifacts"},
		{schema: b.Params, label: "params"},
		{schema: b.Connections, label: "connections"},
		{schema: b.UI, label: "ui"},
	}

	for _, task := range tasks {
		if sch, ok := task.schema.(*Schema); ok {
			setMetadata(sch, task.label, *b)
		}
		content, err := generateSchema(task.schema)

		if err != nil {
			return err
		}

		filepath := fmt.Sprintf("/schema-%s.json", task.label)

		err = os.WriteFile(path.Join(buildPath, filepath), content, 0644)

		if err != nil {
			return err
		}
	}

	return nil
}

// generateSchema generates a specific *-schema.json file
func generateSchema(schema interface{}) ([]byte, error) {
	json, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s\n", string(json))), nil
}

func generateIDURL(mdName string, schemaType string) string {
	return fmt.Sprintf(idURLPattern, mdName, schemaType)
}

func generateSchemaURL(schema string) string {
	return fmt.Sprintf(jsonSchemaURLPattern, schema)
}

// Metadata returns common metadata fields for each JSON Schema
func setMetadata(sch *Schema, schemaType string, b Bundle) {
	sch.Version = generateSchemaURL(b.Schema)
	sch.ID = generateIDURL(b.Name, schemaType)
	sch.Title = b.Name
	sch.Description = b.Description
}
