// Selectable artifact definition table
package artdeftable

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/api"
)

type Model struct {
	table                       table.Model
	help                        help.Model
	artifactDefinitions         []*api.ArtifactDefinition
	keys                        KeyMap
	SelectedArtifactDefinitions []*api.ArtifactDefinition
}

const (
	columnKeyLabel      = "label"
	columnKeyArtDefData = "artDefData"
)

func New(creds []*api.ArtifactDefinition) *Model {
	columns := []table.Column{
		table.NewColumn(columnKeyLabel, "Name", 40),
	}

	rows := []table.Row{}

	for _, credentialType := range creds {
		row := table.NewRow(table.RowData{
			columnKeyLabel:      humanize(credentialType.Name),
			columnKeyArtDefData: credentialType,
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

		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}

	return &Model{
		table:               t,
		help:                help.New(),
		artifactDefinitions: creds,
		keys:                keys,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	m.table, cmd = m.table.Update(msg)
	m.SelectedArtifactDefinitions = mapRowsToArtDef(m.table.SelectedRows())
	return m, cmd
}

func (m Model) View() string {
	body := strings.Builder{}
	body.WriteString("Select credential types:")
	body.WriteString("\n")
	body.WriteString(m.table.View())
	body.WriteString("\n")
	body.WriteString(m.help.View(m.keys))
	return body.String()
}

func mapRowsToArtDef(rows []table.Row) []*api.ArtifactDefinition {
	artdefs := []*api.ArtifactDefinition{}

	for _, row := range rows {
		artdef := row.Data[columnKeyArtDefData].(*api.ArtifactDefinition)
		artdefs = append(artdefs, artdef)
	}
	return artdefs
}

func humanize(artdef string) string {
	abbrevMap := map[string]string{
		"iam": "IAM",
		"gcp": "GCP",
		"aws": "AWS",
	}

	components := strings.Split(artdef, "/")
	friendlyComponents := strings.Split(components[1], "-")

	titledComponents := []string{}

	for _, c := range friendlyComponents {
		var word string
		if v, ok := abbrevMap[c]; ok {
			word = v
		} else {
			word = strings.Title(c)
		}

		titledComponents = append(titledComponents, word)
	}

	return strings.Join(titledComponents, " ")
}
