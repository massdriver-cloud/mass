// Package resourcetypetable provides a selectable resource type table TUI component.
package resourcetypetable

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/api/v0"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Model is the bubbletea model for the selectable resource type table component.
type Model struct {
	table                 table.Model
	help                  help.Model
	resourceTypes         []*api.ArtifactDefinition
	keys                  KeyMap
	SelectedResourceTypes []*api.ArtifactDefinition
}

const (
	columnKeyLabel            = "label"
	columnKeyResourceTypeData = "resourceTypeData"
)

// New creates a new Model pre-populated with the provided resource types.
func New(creds []*api.ArtifactDefinition) Model {
	columns := []table.Column{
		table.NewColumn(columnKeyLabel, "Name", 40),
	}

	rows := []table.Row{}

	for _, credentialType := range creds {
		row := table.NewRow(table.RowData{
			columnKeyLabel:            humanize(credentialType.Name),
			columnKeyResourceTypeData: credentialType,
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
		table:         t,
		help:          help.New(),
		resourceTypes: creds,
		keys:          keys,
	}
}

// Init returns the initial command for the resourcetypetable model (none required).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the resourcetypetable model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//nolint:gocritic // single-case type switch is intentional; msg is reused as typed value below
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.help.ShowAll = true
	}

	m.table, cmd = m.table.Update(msg)
	m.SelectedResourceTypes = mapRowsToResourceType(m.table.SelectedRows())
	return m, cmd
}

// View renders the resource type table as a string for display.
func (m Model) View() string {
	body := strings.Builder{}
	body.WriteString("Select credential types:")
	body.WriteString("\n")
	body.WriteString(m.table.View())
	body.WriteString("\n")
	body.WriteString(m.help.View(m.keys))
	return body.String()
}

func mapRowsToResourceType(rows []table.Row) []*api.ArtifactDefinition {
	resourceTypes := []*api.ArtifactDefinition{}

	for _, row := range rows {
		if rt, ok := row.Data[columnKeyResourceTypeData].(*api.ArtifactDefinition); ok {
			resourceTypes = append(resourceTypes, rt)
		}
	}
	return resourceTypes
}

func humanize(name string) string {
	abbrevMap := map[string]string{
		"iam": "IAM",
		"gcp": "GCP",
		"aws": "AWS",
	}

	components := strings.Split(name, "/")
	friendlyComponents := strings.Split(components[1], "-")

	titledComponents := []string{}

	for _, c := range friendlyComponents {
		var word string
		if v, ok := abbrevMap[c]; ok {
			word = v
		} else {
			word = cases.Title(language.English).String(c)
		}

		titledComponents = append(titledComponents, word)
	}

	return strings.Join(titledComponents, " ")
}
