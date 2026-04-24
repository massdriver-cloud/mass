package resourcetype

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

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// DereferenceOptions holds configuration for schema dereferencing operations.
type DereferenceOptions struct {
	Client  *client.Client
	Cwd     string
	StripID bool
}

// relativeFilePathPattern only accepts relative file path prefixes "./" and "../"
var relativeFilePathPattern = regexp.MustCompile(`^(\.\/|\.\.\/)`)
var massdriverResourceTypePattern = regexp.MustCompile(`^[a-zA-Z0-9-]+(\/[a-zA-Z0-9-]+)?$`)
var httpPattern = regexp.MustCompile(`^(http|https)://`)
var fragmentPattern = regexp.MustCompile(`^#`)

// DereferenceSchema recursively resolves $ref pointers in a schema value.
func DereferenceSchema(anyVal any, opts DereferenceOptions) (any, error) {
	val := getValue(anyVal)

	switch val.Kind() { //nolint:exhaustive // only slice/array and map need dereferencing; other kinds returned as-is
	case reflect.Slice, reflect.Array:
		return dereferenceList(val, opts)
	case reflect.Map:
		schemaInterface := val.Interface()
		schema, ok := schemaInterface.(map[string]any)
		if !ok {
			return nil, errors.New("schema is not an object")
		}
		hydratedSchema := map[string]any{}

		// if this part of the schema has a $ref that is a local file, read it and make it
		// the map that we hydrate into. This causes any keys in the ref'ing object to override anything in the ref'd object
		// which adheres to the JSON Schema spec.
		if schemaRefInterface, refOk := schema["$ref"]; refOk {
			schemaRefValue, refStringOk := schemaRefInterface.(string)
			if !refStringOk {
				return nil, errors.New("$ref is not a string")
			}

			var err error
			if relativeFilePathPattern.MatchString(schemaRefValue) { //nolint:gocritic // long if-else chain matches ref types; restructuring reduces readability
				// this is a relative file ref
				// build up the path from where the dir current schema was read
				hydratedSchema, err = dereferenceFilePathRef(hydratedSchema, schema, schemaRefValue, opts)
			} else if httpPattern.MatchString(schemaRefValue) {
				// HTTP ref. Pull the schema down via HTTP GET and hydrate
				hydratedSchema, err = dereferenceHTTPRef(hydratedSchema, schema, schemaRefValue, opts)
			} else if massdriverResourceTypePattern.MatchString(schemaRefValue) {
				// this must be a published schema, so fetch from massdriver
				hydratedSchema, err = dereferenceMassdriverRef(hydratedSchema, schema, schemaRefValue, opts)
			} else if fragmentPattern.MatchString(schemaRefValue) { //nolint:revive // fragment refs are intentionally left as-is
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
	for i := range val.Len() {
		hydratedVal, err := DereferenceSchema(val.Index(i).Interface(), opts)
		if err != nil {
			return hydratedList, err
		}
		hydratedList = append(hydratedList, hydratedVal)
	}
	return hydratedList, nil
}

func dereferenceMassdriverRef(hydratedSchema map[string]any, schema map[string]any, schemaRefValue string, opts DereferenceOptions) (map[string]any, error) {
	referencedSchema, err := GetAsMap(context.Background(), opts.Client, schemaRefValue)
	if err != nil {
		return hydratedSchema, err
	}

	if nestedSchema, exists := referencedSchema["schema"]; exists {
		var ok bool
		referencedSchema, ok = nestedSchema.(map[string]any)
		if !ok {
			return hydratedSchema, errors.New("schema is not a map")
		}
	}

	// This is a hack to get around the issue of the UI choking if the params schema has 2 or more of the same $id in it.
	// We need the "$id" in resources and connections, but we need to strip it out of params and ui schemas, hence the conditional.
	// This logic should be removed when we have a better solution for this in the UI/API - probably after resource types are in OCI
	if opts.StripID {
		delete(referencedSchema, "$id")
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
