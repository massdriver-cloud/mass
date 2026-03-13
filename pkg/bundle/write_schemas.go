package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

const idURLPattern = "https://schemas.massdriver.cloud/schemas/bundles/%s/schema-%s.json"
const jsonSchemaURL = "http://json-schema.org/draft-07/schema"

// Schema holds a JSON schema map and its label used when writing schema files.
type Schema struct {
	schema map[string]any
	label  string
}

// WriteSchemas writes the bundle's artifact, params, connections, and UI schemas to JSON files in buildPath.
func (b *Bundle) WriteSchemas(buildPath string) error {
	mkdirErr := os.MkdirAll(buildPath, 0750)

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

		filepath := fmt.Sprintf("/schema-%s.json", task.label)

		// #nosec G306
		err = os.WriteFile(path.Join(buildPath, filepath), content, 0644)

		if err != nil {
			return err
		}
	}

	return nil
}

// generateSchema generates a specific *-schema.json file
func generateSchema(schema map[string]any, metadata map[string]string) ([]byte, error) {
	var err error
	var mergedSchema = mergeMaps(schema, metadata)

	json, err := json.MarshalIndent(mergedSchema, "", "    ")
	if err != nil {
		return nil, err
	}

	return []byte(string(json) + "\n"), nil
}

func mergeMaps(a map[string]any, b map[string]string) map[string]any {
	for k, v := range b {
		a[k] = v
	}

	return a
}

func generateIDURL(mdName string, schemaType string) string {
	return fmt.Sprintf(idURLPattern, mdName, schemaType)
}

// buildMetadata returns common metadata fields for each JSON Schema
func buildMetadata(schemaType string, b Bundle) map[string]string {
	if schemaType == "ui" {
		return make(map[string]string)
	}

	return map[string]string{
		"$schema":     jsonSchemaURL,
		"$id":         generateIDURL(b.Name, schemaType),
		"title":       b.Name,
		"description": b.Description,
	}
}
