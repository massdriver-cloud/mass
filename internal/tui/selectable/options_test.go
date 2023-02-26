package selectable_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/tui/selectable"
)

func TestOptionsWithTitle(t *testing.T) {
	columns := []table.Column{}
	model := selectable.New(columns).WithTitle("My Table")

	got := model.View()
	want := "My Table"

	if !strings.Contains(got, want) {
		t.Errorf("Expected to include:\n%s\nGot:\n%s", want, got)
	}
}

func TestOptionsWithRows(t *testing.T) {
	nameColumn := "name"
	columns := []table.Column{
		table.NewColumn(nameColumn, "Name", 20),
	}

	rows := []table.Row{
		table.NewRow(table.RowData{nameColumn: "Chauncy"}),
		table.NewRow(table.RowData{nameColumn: "Buddy"}),
	}

	got := selectable.
		New(columns).
		WithRows(rows).
		Focused(true).
		View()

	if !strings.Contains(got, "Chauncy") || !strings.Contains(got, "Buddy") {
		t.Errorf("expected row values to render, got:\n%s", got)
	}
}

func TestOptionsWithMinimum(t *testing.T) {
	people := []person{
		{name: "Chauncy", id: "1234567890"},
		{name: "Buddy", id: "987654321"},
	}

	// Focus to make it interactive
	model := newPeopleTable(people).Focused(true).WithMinimum(2)

	pressSave := tea.KeyMsg{Type: tea.KeyRunes, Alt: false, Runes: []rune{'s'}}
	updatedModel, _ := model.Update(pressSave)

	view := updatedModel.View()

	if !strings.Contains(view, "You must select at least 2 record(s).") {
		t.Errorf("expected validation error, got:\n%s\n", view)
	}
}

func TestOptionsWithMaximum(t *testing.T) {
	people := []person{
		{name: "Chauncy", id: "1234567890"},
		{name: "Buddy", id: "987654321"},
	}

	// Focus to make it interactive
	model := newPeopleTable(people).Focused(true).WithMaximum(1)

	select1 := tea.KeyMsg{Type: tea.KeyEnter, Alt: false, Runes: []rune{}}
	updatedModel, _ := model.Update(select1)

	moveCursorDown := tea.KeyMsg{Type: tea.KeyRunes, Alt: false, Runes: []rune{'j'}}
	updatedModel, _ = updatedModel.Update(moveCursorDown)

	select2 := tea.KeyMsg{Type: tea.KeyEnter, Alt: false, Runes: []rune{}}
	updatedModel, _ = updatedModel.Update(select2)

	pressSave := tea.KeyMsg{Type: tea.KeyRunes, Alt: false, Runes: []rune{'s'}}
	updatedModel, _ = updatedModel.Update(pressSave)

	view := updatedModel.View()

	if !strings.Contains(view, "You must select at most 1 record(s).") {
		t.Errorf("expected validation error, got:\n%s\n", view)
	}
}
