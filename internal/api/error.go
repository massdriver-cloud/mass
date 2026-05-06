package api

import (
	"fmt"
)

// MutationError represents an error returned from a GraphQL mutation, including validation messages.
type MutationError struct {
	Err      string
	Messages []MutationValidationError
}

// MutationValidationError represents a single validation error from a mutation.
type MutationValidationError struct {
	Code    string
	Field   string
	Message string
}

func (m *MutationError) Error() string {
	err := fmt.Sprintf("GraphQL mutation %s\n", m.Err)

	for _, msg := range m.Messages {
		err = fmt.Sprintf("%s  - %s\n", err, msg.Message)
	}

	return err
}

// NewMutationError creates a new MutationError with the given message and validation errors.
func NewMutationError(msg string, validationErrors []MutationValidationError) error {
	return &MutationError{Err: msg, Messages: validationErrors}
}
