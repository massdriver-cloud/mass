package jsonschema

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
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

type DereferenceOptions struct {
	Client *restclient.MassdriverClient
	Cwd    string
}

// relativeFilePathPattern only accepts relative file path prefixes "./" and "../"
var relativeFilePathPattern = regexp.MustCompile(`^(\.\/|\.\.\/)`)
var massdriverDefinitionPattern = regexp.MustCompile(`^[a-zA-Z0-9]`)
var httpPattern = regexp.MustCompile(`^(http|https)://`)

func Dereference(anyVal interface{}, opts DereferenceOptions) (interface{}, error) {
	val := getValue(anyVal)

	switch val.Kind() { //nolint:exhaustive
	case reflect.Slice, reflect.Array:
		return dereferenceList(val, opts)
	case reflect.Map:
		schemaInterface := val.Interface()
		schema, ok := schemaInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("schema is not an object")
		}
		hydratedSchema := map[string]interface{}{}

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

func dereferenceMap(hydratedSchema map[string]interface{}, schema map[string]interface{}, opts DereferenceOptions) (map[string]interface{}, error) {
	for key, value := range schema {
		var valueInterface = value
		hydratedValue, err := Dereference(valueInterface, opts)
		if err != nil {
			return hydratedSchema, err
		}
		hydratedSchema[key] = hydratedValue
	}
	return hydratedSchema, nil
}

func dereferenceList(val reflect.Value, opts DereferenceOptions) ([]interface{}, error) {
	hydratedList := make([]interface{}, 0)
	for i := 0; i < val.Len(); i++ {
		hydratedVal, err := Dereference(val.Index(i).Interface(), opts)
		if err != nil {
			return hydratedList, err
		}
		hydratedList = append(hydratedList, hydratedVal)
	}
	return hydratedList, nil
}

func dereferenceMassdriverRef(hydratedSchema map[string]interface{}, schema map[string]interface{}, schemaRefValue string, opts DereferenceOptions) (map[string]interface{}, error) {
	referencedSchema, err := definition.Get(opts.Client, schemaRefValue)
	if err != nil {
		return hydratedSchema, err
	}

	if nestedSchema, exists := referencedSchema["schema"]; exists {
		var ok bool
		referencedSchema, ok = nestedSchema.(map[string]interface{})
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

func dereferenceHTTPRef(hydratedSchema map[string]interface{}, schema map[string]interface{}, schemaRefValue string, opts DereferenceOptions) (map[string]interface{}, error) {
	ctx := context.Background()
	var referencedSchema map[string]interface{}
	request, err := http.NewRequestWithContext(ctx, "GET", schemaRefValue, nil)
	if err != nil {
		return hydratedSchema, err
	}
	resp, doErr := opts.Client.Client.Do(request)
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

func dereferenceFilePathRef(hydratedSchema map[string]interface{}, schema map[string]interface{}, schemaRefValue string, opts DereferenceOptions) (map[string]interface{}, error) {
	var referencedSchema map[string]interface{}
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

func getValue(anyVal interface{}) reflect.Value {
	val := reflect.ValueOf(anyVal)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val
}

func readJSONFile(filepath string) (map[string]interface{}, error) {
	var result map[string]interface{}
	data, err := os.ReadFile(filepath)

	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)

	return result, err
}

func replaceRef(base map[string]interface{}, referenced map[string]interface{}, opts DereferenceOptions) (map[string]interface{}, error) {
	hydratedSchema := map[string]interface{}{}
	delete(base, "$ref")

	for k, v := range referenced {
		hydratedValue, err := Dereference(v, opts)
		if err != nil {
			return hydratedSchema, err
		}
		hydratedSchema[k] = hydratedValue
	}
	return hydratedSchema, nil
}
