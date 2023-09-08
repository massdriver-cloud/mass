package jsonschema

import (
	"github.com/xeipuuv/gojsonschema"
)

// Validate the input object against the schema
func Validate(schemaPath string, documentPath string) (*gojsonschema.Result, error) {
	sl := Loader(schemaPath)
	dl := Loader(documentPath)

	return gojsonschema.Validate(sl, dl)
}
