package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

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

		dereferencedSchema, err := DereferenceSchema(*task.schema, DereferenceOptions{Client: mdClient, Cwd: cwd})

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

type DereferenceOptions struct {
	Client *client.Client
	Cwd    string
}

// relativeFilePathPattern only accepts relative file path prefixes "./" and "../"
var relativeFilePathPattern = regexp.MustCompile(`^(\.\/|\.\.\/)`)
var massdriverDefinitionPattern = regexp.MustCompile(`^[a-zA-Z0-9]`)
var httpPattern = regexp.MustCompile(`^(http|https)://`)

func DereferenceSchema(anyVal any, opts DereferenceOptions) (any, error) {
	val := getValue(anyVal)

	switch val.Kind() { //nolint:exhaustive
	case reflect.Slice, reflect.Array:
		return dereferenceList(val, opts)
	case reflect.Map:
		schemaInterface := val.Interface()
		schema, ok := schemaInterface.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema is not an object")
		}
		hydratedSchema := map[string]any{}

		// if this part of the schema has a $ref that is a local file, read it and make it
		// the map that we hydrate into. This causes any keys in the ref'ing object to override anything in the ref'd object
		// which adheres to the JSON Schema spec.
		if schemaRefInterface, refOk := schema["$ref"]; refOk {
			schemaRefValue, refStringOk := schemaRefInterface.(string)
			if !refStringOk {
				return nil, fmt.Errorf("$ref is not a string")
			}

			var err error
			if relativeFilePathPattern.MatchString(schemaRefValue) { //nolint:gocritic
				// this is a local file ref
				// build up the path from where the dir current schema was read
				hydratedSchema, err = dereferenceFilePathRef(hydratedSchema, schema, schemaRefValue, opts)
			} else if httpPattern.MatchString(schemaRefValue) {
				// HTTP ref. Pull the schema down via HTTP GET and hydrate
				hydratedSchema, err = dereferenceHTTPRef(hydratedSchema, schema, schemaRefValue, opts)
			} else if massdriverDefinitionPattern.MatchString(schemaRefValue) {
				// this must be a published schema, so fetch from massdriver
				hydratedSchema, err = dereferenceMassdriverRef(hydratedSchema, schema, schemaRefValue, opts)
			} else {
				return nil, fmt.Errorf("unable to resolve ref: %s", schemaRefValue)
			}
			if err != nil {
				return hydratedSchema, err
			}
		}
		return dereferenceMap(hydratedSchema, schema, opts)
	default:
		return anyVal, nil
	}
}

func dereferenceMap(hydratedSchema map[string]any, schema map[string]any, opts DereferenceOptions) (map[string]any, error) {
	for key, value := range schema {
		var valueInterface = value
		hydratedValue, err := DereferenceSchema(valueInterface, opts)
		if err != nil {
			return hydratedSchema, err
		}
		hydratedSchema[key] = hydratedValue
	}
	return hydratedSchema, nil
}

func dereferenceList(val reflect.Value, opts DereferenceOptions) ([]any, error) {
	hydratedList := make([]any, 0)
	for i := 0; i < val.Len(); i++ {
		hydratedVal, err := DereferenceSchema(val.Index(i).Interface(), opts)
		if err != nil {
			return hydratedList, err
		}
		hydratedList = append(hydratedList, hydratedVal)
	}
	return hydratedList, nil
}

func dereferenceMassdriverRef(hydratedSchema map[string]any, schema map[string]any, schemaRefValue string, opts DereferenceOptions) (map[string]any, error) {
	referencedSchema, err := definition.GetAsMap(context.Background(), opts.Client, schemaRefValue)
	if err != nil {
		return hydratedSchema, err
	}

	if nestedSchema, exists := referencedSchema["schema"]; exists {
		var ok bool
		referencedSchema, ok = nestedSchema.(map[string]any)
		if !ok {
			return hydratedSchema, fmt.Errorf("schema is not a map")
		}
	}

	hydratedSchema, err = replaceRef(schema, referencedSchema, opts)
	if err != nil {
		return hydratedSchema, err
	}
	return hydratedSchema, nil
}

func dereferenceHTTPRef(hydratedSchema map[string]any, schema map[string]any, schemaRefValue string, opts DereferenceOptions) (map[string]any, error) {
	ctx := context.Background()
	var referencedSchema map[string]any

	client := http.Client{}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, schemaRefValue, nil)
	if err != nil {
		return hydratedSchema, err
	}
	resp, doErr := client.Do(request)
	if doErr != nil {
		return hydratedSchema, doErr
	}
	if resp.StatusCode != http.StatusOK {
		return hydratedSchema, errors.New("received non-200 response getting ref " + resp.Status + " " + schemaRefValue)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return hydratedSchema, err
	}
	err = json.Unmarshal(body, &referencedSchema)
	if err != nil {
		return hydratedSchema, err
	}

	hydratedSchema, err = replaceRef(schema, referencedSchema, opts)
	return hydratedSchema, err
}

func dereferenceFilePathRef(hydratedSchema map[string]any, schema map[string]any, schemaRefValue string, opts DereferenceOptions) (map[string]any, error) {
	var referencedSchema map[string]any
	var schemaRefDir string
	schemaRefAbsPath, err := filepath.Abs(filepath.Join(opts.Cwd, schemaRefValue))

	if err != nil {
		return hydratedSchema, err
	}

	schemaRefDir = filepath.Dir(schemaRefAbsPath)
	referencedSchema, readErr := readJSONFile(schemaRefAbsPath)

	if readErr != nil {
		return hydratedSchema, readErr
	}

	var replaceErr error
	opts.Cwd = schemaRefDir
	hydratedSchema, replaceErr = replaceRef(schema, referencedSchema, opts)
	if replaceErr != nil {
		return hydratedSchema, replaceErr
	}
	return hydratedSchema, nil
}

func getValue(anyVal any) reflect.Value {
	val := reflect.ValueOf(anyVal)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val
}

func readJSONFile(filepath string) (map[string]any, error) {
	var result map[string]any
	data, err := os.ReadFile(filepath)

	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)

	return result, err
}

func replaceRef(base map[string]any, referenced map[string]any, opts DereferenceOptions) (map[string]any, error) {
	hydratedSchema := map[string]any{}
	delete(base, "$ref")

	for k, v := range referenced {
		hydratedValue, err := DereferenceSchema(v, opts)
		if err != nil {
			return hydratedSchema, err
		}
		hydratedSchema[k] = hydratedValue
	}
	return hydratedSchema, nil
}
