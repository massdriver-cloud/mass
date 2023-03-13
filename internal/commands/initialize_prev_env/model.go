package initializeprevenv

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
	"github.com/massdriver-cloud/mass/internal/tui/components/artifacttable"
)

const (
	artifactDefinitionTable int = 0
)

type KeyMap struct {
	Quit key.Binding
	Next key.Binding
	Back key.Binding
}

type Model struct {
	screens                   []tea.Model
	currentScreen             int
	keys                      KeyMap
	quitting                  bool
	loaded                    bool
	ListCredentials           func(string) ([]*api.Artifact, error)
	SelectedCredentialsByType map[string]string
}

func New() *Model {
	keys := KeyMap{
		Next: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next"),
		),
		Back: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
	return &Model{
		keys: keys,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if !m.loaded {
		return "loading..."
	}

	if m.quitting {
		return ""
	}

	view := m.screens[m.currentScreen].View()
	// TODO: return keys and display combined help + back/next (esc issue)
	return fmt.Sprintf("%s\n\n", view)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.currentScreen = artifactDefinitionTable
			m.loaded = true
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Next):
			if m.currentScreen == artifactDefinitionTable {
				m.screens = buildscreens(m)
			}

			m.currentScreen++
		case key.Matches(msg, m.keys.Back):
			if m.currentScreen > 0 {
				m.currentScreen--
			}
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	if m.currentScreen == len(m.screens) {
		m.quitting = true
		return m, tea.Quit
	} else {
		m.screens[m.currentScreen], cmd = m.screens[m.currentScreen].Update(msg)
		return m, cmd
	}
}

func buildscreens(m Model) []tea.Model {
	screens := []tea.Model{m.screens[0]}

	if v, ok := m.screens[artifactDefinitionTable].(artdeftable.Model); ok {
		for _, artdef := range v.SelectedArtifactDefinitions {
			screens = append(screens, buildArtifactTable(m, artdef.Name))
		}
	}

	return screens
}

func buildArtifactTable(m Model, artdefName string) *artifacttable.Model {
	creds, _ := m.ListCredentials(artdefName)
	return artifacttable.New(creds)
}
