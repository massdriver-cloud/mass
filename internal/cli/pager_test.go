package cli

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestIsInteractiveFalseForFile(t *testing.T) {
	if IsInteractive(tempFile(t)) {
		t.Error("a regular file should not be reported as interactive")
	}
	if IsInteractive(nil) {
		t.Error("nil file should not be reported as interactive")
	}
}
