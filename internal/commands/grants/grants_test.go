package grants_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/grants"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestRenderTable(t *testing.T) {
	grant := types.Grant{
		ID:                  "grant-1",
		Action:              grants.ActionRepoPull,
		RecipientConditions: types.PolicyConditions{"team": {"platform", "data"}},
		CreatedAt:           time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC),
	}

	out, err := renderTo(t, func(out *os.File) error {
		return grants.Render(grantSeq(grant), "table", out)
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, want := range []string{"grant-1", grants.ActionRepoPull, "team=platform|data", "2026-09-10 12:30:00"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderTableNamesTheWildcard(t *testing.T) {
	out, err := renderTo(t, func(out *os.File) error {
		return grants.Render(grantSeq(types.Grant{ID: "grant-1", Action: grants.ActionRepoPull}), "table", out)
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(out, "everyone") {
		t.Errorf("table output missing %q:\n%s", "everyone", out)
	}
}

func TestRenderJSONConditionsRoundTripThroughFile(t *testing.T) {
	want := types.PolicyConditions{"team": {"platform", "data"}, "region": nil}

	out, err := renderTo(t, func(out *os.File) error {
		return grants.Render(grantSeq(types.Grant{ID: "grant-1", RecipientConditions: want}), "json", out)
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	var rendered []struct {
		RecipientConditions json.RawMessage `json:"recipientConditions"`
	}
	if unmarshalErr := json.Unmarshal([]byte(out), &rendered); unmarshalErr != nil {
		t.Fatalf("rendered JSON did not parse: %v\n%s", unmarshalErr, out)
	}
	if len(rendered) != 1 {
		t.Fatalf("rendered %d grants, want 1:\n%s", len(rendered), out)
	}

	file := filepath.Join(t.TempDir(), "conditions.json")
	if writeErr := os.WriteFile(file, rendered[0].RecipientConditions, 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	got, err := grants.ParseConditionsFile(file, "all-projects")
	if err != nil {
		t.Fatalf("ParseConditionsFile on rendered conditions returned error: %v", err)
	}
	if !conditionsEqual(got, want) {
		t.Errorf("round trip = %v, want %v", got, want)
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	err := grants.Render(grantSeq(), "yaml", os.Stdout)
	if err == nil {
		t.Fatal("Render succeeded, wanted error")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error = %q, want it to name the unsupported format", err)
	}
}

func TestRenderPropagatesIterationError(t *testing.T) {
	pageErr := errors.New("page fetch failed")

	for _, format := range []string{"table", "json"} {
		t.Run(format, func(t *testing.T) {
			_, err := renderTo(t, func(out *os.File) error {
				return grants.Render(failingSeq(pageErr), format, out)
			})
			if !errors.Is(err, pageErr) {
				t.Errorf("Render error = %v, want %v", err, pageErr)
			}
		})
	}
}

func TestValidateGrantID(t *testing.T) {
	if err := grants.ValidateGrantID("4f2a9c18-1f0e-4a5c-9a3e-2b6d7c8e9f10", "mass repository grant list <name>"); err != nil {
		t.Errorf("ValidateGrantID rejected a valid uuid: %v", err)
	}

	for _, id := range []string{"not-a-uuid", "123", "", "4f2a9c18-1f0e-4a5c-9a3e-2b6d7c8e9f1", "4f2a9c18_1f0e_4a5c_9a3e_2b6d7c8e9f10"} {
		t.Run(id, func(t *testing.T) {
			err := grants.ValidateGrantID(id, "mass repository grant list <name>")
			if err == nil {
				t.Fatalf("ValidateGrantID(%q) succeeded, wanted error", id)
			}
			if !strings.Contains(err.Error(), "mass repository grant list") {
				t.Errorf("error = %q, want it to name the list command", err)
			}
		})
	}
}

func TestNotFoundHint(t *testing.T) {
	err := grants.NotFoundHint(gql.ErrNotFound, "repository", "aws-aurora-postgres")
	if err == nil {
		t.Fatal("NotFoundHint dropped the error")
	}
	for _, want := range []string{"repository", "aws-aurora-postgres", "does not exist"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestNotFoundHintUnwrapsWrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("list oci repo grants: %w", gql.ErrNotFound)
	err := grants.NotFoundHint(wrapped, "repository", "aws-aurora-postgres")
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want the not-found rewrite", err)
	}
}

func TestNotFoundHintPassesOtherErrorsThrough(t *testing.T) {
	original := errors.New("permission denied")
	if err := grants.NotFoundHint(original, "repository", "aws-aurora-postgres"); !errors.Is(err, original) {
		t.Errorf("NotFoundHint = %v, want %v", err, original)
	}
}

func TestNotFoundHintNilStaysNil(t *testing.T) {
	if err := grants.NotFoundHint(nil, "repository", "aws-aurora-postgres"); err != nil {
		t.Errorf("NotFoundHint(nil) = %v, want nil", err)
	}
}

// A plain file is not a TTY, so the table path streams instead of paging.
func renderTo(t *testing.T, fn func(out *os.File) error) (string, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "render.out")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	runErr := fn(file)

	if closeErr := file.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	out, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}

	return string(out), runErr
}

func grantSeq(items ...types.Grant) iter.Seq2[types.Grant, error] {
	return func(yield func(types.Grant, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func failingSeq(err error) iter.Seq2[types.Grant, error] {
	return func(yield func(types.Grant, error) bool) {
		yield(types.Grant{}, err)
	}
}
