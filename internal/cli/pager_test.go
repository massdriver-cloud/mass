package cli

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
)

type pagerRow struct {
	id   string
	name string
}

// seqOf builds an iter.Seq2 over items. If errAt >= 0, it yields an error in
// place of the item at that index and stops, mimicking a failed page fetch.
func seqOf(items []pagerRow, errAt int) iter.Seq2[pagerRow, error] {
	return func(yield func(pagerRow, error) bool) {
		for i, it := range items {
			if i == errAt {
				yield(pagerRow{}, errors.New("page fetch failed"))
				return
			}
			if !yield(it, nil) {
				return
			}
		}
	}
}

func pagerConfig(out *os.File) PagerConfig[pagerRow] {
	return PagerConfig[pagerRow]{
		Columns: []string{"ID", "Name"},
		Row:     func(r pagerRow) []string { return []string{r.id, r.name} },
		Out:     out,
	}
}

// tempFile gives a non-terminal *os.File so Paginate takes the streamAll path.
func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out.txt"))
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestPaginateNonInteractiveStreamsAllRows(t *testing.T) {
	out := tempFile(t)
	items := []pagerRow{
		{"proj-1", "Alpha"},
		{"proj-2", "Bravo"},
		{"proj-3", "Charlie"},
	}

	if err := Paginate(seqOf(items, -1), pagerConfig(out)); err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}

	got, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	rendered := string(got)
	for _, want := range []string{"ID", "Name", "proj-1", "Alpha", "proj-3", "Charlie"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, rendered)
		}
	}
}

func TestPaginateNonInteractivePropagatesError(t *testing.T) {
	out := tempFile(t)
	items := []pagerRow{{"proj-1", "Alpha"}, {"proj-2", "Bravo"}}

	err := Paginate(seqOf(items, 1), pagerConfig(out))
	if err == nil {
		t.Fatal("expected error from failed page fetch, got nil")
	}
	if !strings.Contains(err.Error(), "page fetch failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTruncateRowsClipsWideCellsAndPreservesNarrow(t *testing.T) {
	// Column widths include the +2 padding sizeColumns adds, so usable data
	// widths are 4 and 6 here.
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 8},
	}
	rows := []table.Row{
		{"short", "ok"},                            // ID overflows by 1, Name fits
		{"abc", "this name is way too long"},       // Name overflows
		{"fits", "fine"},                           // both fit
	}

	got := truncateRows(rows, cols)
	if len(got) != len(rows) {
		t.Fatalf("expected %d rows, got %d", len(rows), len(got))
	}

	checkWidth := func(t *testing.T, cell string, max int) {
		t.Helper()
		// Each rune of "…" is one cell wide; runewidth.Truncate guarantees
		// output width <= max.
		if w := len([]rune(cell)); w > max {
			t.Errorf("cell %q has rune-width %d, exceeds limit %d", cell, w, max)
		}
	}

	checkWidth(t, got[0][0], 4)
	checkWidth(t, got[1][1], 6)

	if !strings.HasSuffix(got[1][1], "…") {
		t.Errorf("expected overflowed cell to end with ellipsis, got %q", got[1][1])
	}
	if got[2][0] != "fits" || got[2][1] != "fine" {
		t.Errorf("non-overflowing row mutated: %v", got[2])
	}
}

func TestTruncateRowsHandlesNoColumns(t *testing.T) {
	rows := []table.Row{{"a", "b"}}
	got := truncateRows(rows, nil)
	if len(got) != 1 || got[0][0] != "a" || got[0][1] != "b" {
		t.Errorf("expected rows passthrough when cols is nil, got %v", got)
	}
}

func TestIsInteractiveFalseForFile(t *testing.T) {
	if IsInteractive(tempFile(t)) {
		t.Error("a regular file should not be reported as interactive")
	}
	if IsInteractive(nil) {
		t.Error("nil file should not be reported as interactive")
	}
}
