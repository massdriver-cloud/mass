package api

import (
	"fmt"
)

type MutationError struct {
	Err      string
	Messages []MutationValidationError
}

func (m *MutationError) Error() string {
	err := fmt.Sprintf("Error in GraphQL Mutation: %s\n", m.Err)

	for _, msg := range m.Messages {
		err = fmt.Sprintf("%s  - %s\n", err, msg.Message)
	}

	return err
}

func NewMutationError(msg string, validationErrors []MutationValidationError) error {
	return &MutationError{Err: msg, Messages: validationErrors}
}
