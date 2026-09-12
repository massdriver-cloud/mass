// Package grants holds the logic behind `mass repository grant` and
// `mass resource grant`.
package grants

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"regexp"

	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// out doubles as the interactivity signal: a TTY gets the pager, anything
// else a streamed table.
func Render(seq iter.Seq2[types.Grant, error], output string, out *os.File) error {
	switch output {
	case "json":
		items, collectErr := types.Collect(seq)
		if collectErr != nil {
			return collectErr
		}
		jsonBytes, marshalErr := json.MarshalIndent(items, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal grants to JSON: %w", marshalErr)
		}
		fmt.Fprintln(out, string(jsonBytes))
		return nil
	case "table":
		return cli.Paginate(seq, cli.PagerConfig[types.Grant]{
			Out:     out,
			Columns: []string{"ID", "Action", "Recipients", "Created At"},
			Row: func(grant types.Grant) []string {
				return []string{
					grant.ID,
					grant.Action,
					FormatConditions(grant.RecipientConditions),
					grant.CreatedAt.Format("2006-01-02 15:04:05"),
				}
			},
		})
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

var grantIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// The server rejects a non-UUID id with a raw GraphQL parse error, so catch it
// here. listCmd names the command that lists ids.
func ValidateGrantID(id, listCmd string) error {
	if !grantIDPattern.MatchString(id) {
		return fmt.Errorf("%q is not a grant id; run `%s` to find it", id, listCmd)
	}
	return nil
}

func NotFoundHint(err error, kind, id string) error {
	if !errors.Is(err, gql.ErrNotFound) {
		return err
	}
	return fmt.Errorf("%s %q does not exist", kind, id)
}
