package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidateFile validates a document file against a compiled JSON schema.
func ValidateFile(sch *jsonschema.Schema, documentPath string) error {
	document, loadErr := loadFile(documentPath)
	if loadErr != nil {
		return fmt.Errorf("failed to load document file %q: %w", documentPath, loadErr)
	}
	return sch.Validate(document)
}

// ValidateBytes validates document bytes against a compiled JSON schema.
func ValidateBytes(sch *jsonschema.Schema, documentBytes []byte) error {
	document, loadErr := jsonschema.UnmarshalJSON(bytes.NewBuffer(documentBytes))
	if loadErr != nil {
		return fmt.Errorf("failed to unmarshal document bytes: %w", loadErr)
	}
	return sch.Validate(document)
}

// ValidateGo validates a Go object against a compiled JSON schema.
// The object is marshaled to JSON and then validated.
func ValidateGo(sch *jsonschema.Schema, document any) error {
	documentBytes, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal Go object: %w", err)
	}

	doc, err := jsonschema.UnmarshalJSON(bytes.NewBuffer(documentBytes))
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return sch.Validate(doc)
}
