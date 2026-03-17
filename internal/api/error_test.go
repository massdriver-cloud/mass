package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
)

func TestNewMutationError(t *testing.T) {
	messages := []api.MutationValidationError{
		{Message: "boom"},
		{Message: "pow"},
	}

	err := api.NewMutationError("oops", messages)

	got := err.Error()
	want := "GraphQL mutation oops\n  - boom\n  - pow\n"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
