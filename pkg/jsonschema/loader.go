package jsonschema

import (
	"path/filepath"
	"regexp"

	"github.com/xeipuuv/gojsonschema"
)

const filePrefix = "file://"

var loaderPrefixPattern = regexp.MustCompile(`^(file|http|https)://`)

// Load a JSON Schema with or without a path prefix
func Loader(path string) gojsonschema.JSONLoader {
	var ref string
	if loaderPrefixPattern.MatchString(path) {
		ref = path
	} else {
		// gojsonschema has a strange "reference must be canonical" error if the schema path is the current directory
		if filepath.Dir(path) == "." {
			path = "./" + path
		}
		ref = filePrefix + path
	}

	return gojsonschema.NewReferenceLoader(ref)
}
