package bundle

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/spf13/afero"
)

const idURLPattern = "https://schemas.massdriver.cloud/schemas/bundles/%s/schema-%s.json"
const jsonSchemaURLPattern = "http://json-schema.org/%s/schema"

type Schema struct {
	schema map[string]interface{}
	label  string
}

func (b *Bundle) WriteSchemas(buildPath string, fs afero.Fs) error {
	mkdirErr := fs.MkdirAll(buildPath, 0755)

	if mkdirErr != nil {
		return mkdirErr
	}

	tasks := []Schema{
		{schema: b.Artifacts, label: "artifacts"},
		{schema: b.Params, label: "params"},
		{schema: b.Connections, label: "connections"},
		{schema: b.UI, label: "ui"},
	}

	for _, task := range tasks {
		content, err := generateSchema(task.schema, buildMetadata(task.label, *b))

		if err != nil {
			return err
		}

		filepath := fmt.Sprintf("/%s-schema.json", task.label)

		err = afero.WriteFile(fs, path.Join(buildPath, filepath), content, 0755)

		if err != nil {
			return err
		}
	}

	return nil
}

// generateSchema generates a specific *-schema.json file
func generateSchema(schema map[string]interface{}, metadata map[string]string) ([]byte, error) {
	var err error
	var mergedSchema = mergeMaps(schema, metadata)

	json, err := json.MarshalIndent(mergedSchema, "", "    ")
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s\n", string(json))), nil
}

func mergeMaps(a map[string]interface{}, b map[string]string) map[string]interface{} {
	for k, v := range b {
		a[k] = v
	}

	return a
}

func generateIDURL(mdName string, schemaType string) string {
	return fmt.Sprintf(idURLPattern, mdName, schemaType)
}

func generateSchemaURL(schema string) string {
	return fmt.Sprintf(jsonSchemaURLPattern, schema)
}

// Metadata returns common metadata fields for each JSON Schema
func buildMetadata(schemaType string, b Bundle) map[string]string {
	if schemaType == "ui" {
		return make(map[string]string)
	}

	return map[string]string{
		"$schema":     generateSchemaURL(b.Schema),
		"$id":         generateIDURL(b.Name, schemaType),
		"title":       b.Name,
		"description": b.Description,
	}
}
