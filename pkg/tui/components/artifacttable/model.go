package artifacttable

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/pkg/api"
)

type Model struct {
	table             table.Model
	help              help.Model
	artifacts         []*api.Artifact
	keys              KeyMap
	SelectedArtifacts []*api.Artifact
}

const (
	columnKeyName         = "name"
	columnKeyID           = "id"
	columnKeyArtifactData = "artifactData"
)

func New(artifacts []*api.Artifact) Model {
	columns := []table.Column{
		table.NewColumn(columnKeyName, "Name", 40),
		table.NewColumn(columnKeyID, "ID", 40),
	}

	rows := []table.Row{}

	for _, artifact := range artifacts {
		row := table.NewRow(table.RowData{
			columnKeyName:         artifact.Name,
			columnKeyID:           artifact.ID,
			columnKeyArtifactData: artifact,
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
		artifacts: artifacts,
		keys:      keys,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//nolint:gocritic
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.help.ShowAll = true
	}

	m.table, cmd = m.table.Update(msg)
	m.SelectedArtifacts = mapRowsToArtifact(m.table.SelectedRows())
	return m, cmd
}

func (m Model) View() string {
	body := strings.Builder{}
	body.WriteString("Select credentials:")
	body.WriteString("\n")
	body.WriteString(m.table.View())
	body.WriteString("\n")
	body.WriteString(m.help.View(m.keys))
	return body.String()
}

func mapRowsToArtifact(rows []table.Row) []*api.Artifact {
	artifacts := []*api.Artifact{}

	for _, row := range rows {
		if artifact, ok := row.Data[columnKeyArtifactData].(*api.Artifact); ok {
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts
}
