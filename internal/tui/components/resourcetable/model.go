package resourcetable

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/api/v1"
)

// Model is the Bubble Tea model for the resource selection table.
type Model struct {
	table             table.Model
	help              help.Model
	resources         []*api.Resource
	keys              KeyMap
	SelectedResources []*api.Resource
}

const (
	columnKeyName         = "name"
	columnKeyID           = "id"
	columnKeyResourceData = "resourceData"
)

// New creates a resource table model populated with the given resources.
func New(resources []*api.Resource) Model {
	columns := []table.Column{
		table.NewColumn(columnKeyName, "Name", 40),
		table.NewColumn(columnKeyID, "ID", 40),
	}

	rows := []table.Row{}

	for _, resource := range resources {
		row := table.NewRow(table.RowData{
			columnKeyName:         resource.Name,
			columnKeyID:           resource.ID,
			columnKeyResourceData: resource,
		})
		rows = append(rows, row)
	}

	t := table.
		New(columns).
		WithRows(rows).
		SelectableRows(true).
		Focused(true)

	tableKeyMap := t.KeyMap()

	keys := KeyMap{
		RowUp: key.NewBinding(
			key.WithKeys(tableKeyMap.RowUp.Keys()...),
			key.WithHelp("↑/k", "row up"),
		),

		RowDown: key.NewBinding(
			key.WithKeys(tableKeyMap.RowDown.Keys()...),
			key.WithHelp("↓/j", "row down"),
		),

		RowSelectToggle: key.NewBinding(
			key.WithKeys(tableKeyMap.RowSelectToggle.Keys()...),
			key.WithHelp("space/enter", "select row"),
		),
	}

	return Model{
		table:     t,
		help:      help.New(),
		resources: resources,
		keys:      keys,
	}
}

// Init satisfies the tea.Model interface and performs no initialization.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//nolint:gocritic // single-case type switch is intentional; msg is reused as typed value below
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.help.ShowAll = true
	}

	m.table, cmd = m.table.Update(msg)
	m.SelectedResources = mapRowsToResource(m.table.SelectedRows())
	return m, cmd
}

// View renders the resource table and help text as a string.
func (m Model) View() string {
	body := strings.Builder{}
	body.WriteString("Select credentials:")
	body.WriteString("\n")
	body.WriteString(m.table.View())
	body.WriteString("\n")
	body.WriteString(m.help.View(m.keys))
	return body.String()
}

func mapRowsToResource(rows []table.Row) []*api.Resource {
	resources := []*api.Resource{}

	for _, row := range rows {
		if resource, ok := row.Data[columnKeyResourceData].(*api.Resource); ok {
			resources = append(resources, resource)
		}
	}

	return resources
}
