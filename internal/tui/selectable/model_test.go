package selectable_test

import (
	"strings"
	"testing"

	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/tui/selectable"
)

func TestNew(t *testing.T) {
	columns := []table.Column{
		table.NewColumn("c1", "Column 1", 13),
		table.NewColumn("c2", "Column 2", 13),
	}

	model := selectable.New(columns)

	got := model.View()

	if !strings.Contains(got, "Column 1") || !strings.Contains(got, "Column 2") {
		t.Errorf("expected column headers to render, got:\n'%s'", got)
	}
}
