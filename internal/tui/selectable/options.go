package selectable

import "github.com/evertras/bubble-table/table"

func (m Model) WithTitle(title string) Model {
	m.title = title
	return m
}

func (m Model) WithRows(r []table.Row) Model {
	m.table = m.table.WithRows(r)
	return m
}

func (m Model) WithMinimum(count int) Model {
	m.minimum = count
	return m
}

func (m Model) WithMaximum(count int) Model {
	m.maximum = count
	return m
}
