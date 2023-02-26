package selectable_test

import (
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/tui/selectable"
)

type person struct {
	id   string
	name string
}

func newPeopleTable(people []person) selectable.Model {
	nameColumn := "name"
	columns := []table.Column{
		table.NewColumn(nameColumn, "Name", 20),
	}

	rows := []table.Row{}
	for _, person := range people {
		row := table.NewRow(table.RowData{nameColumn: person.name, "id": person.id})
		rows = append(rows, row)
	}

	return selectable.New(columns).WithRows(rows)
}
