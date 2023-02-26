// simple wrapper for selecting results form a 'table' of data...
package selectable

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TODO: styles (title, table, column headings, help)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.validationError = ""
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// msg.String() feels more friendly than mst.Type
		// which `case` matches tea.KeyCtrlC, tea.KeyEsc, etc
		// see inputfield for example
		switch msg.String() {
		case "s": // 'saving'
			numSelected := len(m.table.SelectedRows())

			if m.maximum > 0 && numSelected > m.maximum {
				m.validationError = fmt.Sprintf("You must select at most %d record(s).", m.maximum)
			} else if numSelected < m.minimum {
				m.validationError = fmt.Sprintf("You must select at least %d record(s).", m.minimum)
			} else {
				return m, tea.Quit
			}
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	// Handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		// This clears the screen on quit... do we want to do that?
		// Should we diff between 'quitting' is _any_ quit... but 'cancel' || 'continue'
		// might be nice to clear the screen between 'screens'
		return ""
	}

	body := strings.Builder{}
	body.WriteString(titleComponent(m))
	body.WriteString(m.table.View())
	body.WriteString("\n")
	body.WriteString(helpOrValidationMessagesComponent(m))

	return body.String()
}

func helpOrValidationMessagesComponent(m Model) string {
	msg := ""
	if m.validationError == "" {
		// TODO: replace w/ https://github.com/charmbracelet/bubbletea/blob/master/examples/help/main.go
		// Help view KeyMap
		// Error area (Validation Errors)
		// table may expose its key bindings for help, if so we just need to add "(S)ave"
		msg = "↑/↓: Navigate • esc: Quit • enter: Select • s: Save"
	} else {
		var style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff0000")).Render
		msg = style(m.validationError)
	}

	return msg + "\n"
}

func titleComponent(m Model) string {
	var style = lipgloss.NewStyle().Bold(true).Render
	return style(m.title) + "\n"
}
