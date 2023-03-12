package selectable

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/evertras/bubble-table/table"
)

type errMsg error

type Model struct {
	table           table.Model
	title           string
	validationError string
	minimum         int
	maximum         int
	quitting        bool
	err             errMsg
	help            help.Model
}

func New(columns []table.Column) Model {
	t := table.
		New(columns).
		SelectableRows(true)

	m := Model{
		table: t,
		help:  help.New(),
		title: "Select records:",
	}

	return m
}

func (m Model) Focused(focused bool) Model {
	m.table = m.table.Focused(focused)
	return m
}

func (m Model) SelectedRows() []map[string]interface{} {
	results := []map[string]interface{}{}

	if m.quitting {
		return results
	}

	for _, row := range m.table.SelectedRows() {
		results = append(results, row.Data)
	}

	return results
}
