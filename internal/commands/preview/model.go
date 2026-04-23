package preview

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/tui/components/resourcetable"
	"github.com/massdriver-cloud/mass/internal/tui/components/resourcetypetable"
)

type mode int

const (
	artifactDefinitionSelection mode = iota
	artifactSelection
)

type artifactPrompt struct {
	artifactDefinitionName string
	selection              api.Artifact
}

// Model is the bubbletea model for the interactive preview environment initialization UI.
type Model struct {
	keys            KeyMap
	help            help.Model
	quitting        bool
	loaded          bool
	listCredentials func(string) ([]*api.Artifact, error)
	project         *api.Project

	// ui mode
	mode mode

	// initial artdef selector
	artDefTable tea.Model

	// current model
	current      tea.Model
	prompts      []artifactPrompt
	promptCursor int
}

// PreviewConfig returns the preview environment configuration built from the user's selections.
func (m Model) PreviewConfig() *api.PreviewConfig {
	credentials := make([]api.Credential, 0) // Initialize with empty slice

	for _, p := range m.prompts {
		credentials = append(credentials, api.Credential{ArtifactDefinitionType: p.artifactDefinitionName, ArtifactId: p.selection.ID})
	}

	return &api.PreviewConfig{
		Packages:    m.project.GetDefaultParams(),
		Credentials: credentials,
		ProjectSlug: m.project.Slug,
	}
}

// Init returns the initial command for the preview model (none required).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the preview model state accordingly.
//
//nolint:gocognit
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.loaded {
		m.current = m.artDefTable
		m.mode = artifactDefinitionSelection
		m.loaded = true
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.Next):
			switch m.mode {
			case artifactDefinitionSelection:
				m.prompts = initArtifactPrompts(m)
				m.mode = artifactSelection
			case artifactSelection:
				currentModel, currentModelOk := m.current.(resourcetable.Model)
				if !currentModelOk {
					return m, nil // type mismatch — abort this keypress entirely, do not advance cursor
				}
				selectedArtifacts := currentModel.SelectedResources
				if len(selectedArtifacts) > 0 {
					// TODO limit 1 in UI w/ Maximum validation error OR call next automatically
					// when selecting in an artifact prompt
					prompt := m.prompts[m.promptCursor]
					prompt.selection = *selectedArtifacts[0]
					m.prompts[m.promptCursor] = prompt
				}
			}

			m.promptCursor++
			if m.promptCursor < len(m.prompts) {
				nextPrompt := m.prompts[m.promptCursor]
				m.current = buildArtifactTable(m, nextPrompt.artifactDefinitionName)
			}

		case key.Matches(msg, m.keys.Back):
			if m.mode == artifactSelection {
				m.promptCursor--
				if m.promptCursor == -1 {
					m.mode = artifactDefinitionSelection
					m.current = m.artDefTable
					m.promptCursor = -1
					return m, cmd
				}

				prevPrompt := m.prompts[m.promptCursor]
				m.current = buildArtifactTable(m, prevPrompt.artifactDefinitionName)
			}

		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.promptCursor == len(m.prompts) {
		m.quitting = true
		return m, tea.Quit
	}

	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

// View renders the current state of the preview model as a string for display.
func (m Model) View() string {
	if !m.loaded {
		return "loading..."
	}

	if m.quitting {
		return ""
	}

	body := strings.Builder{}
	body.WriteString(m.current.View())
	body.WriteString("\n")
	// TODO: combine submodel help w/ nav help
	body.WriteString(m.help.View(m.keys))
	return body.String()
}

func initArtifactPrompts(m Model) []artifactPrompt {
	prompts := []artifactPrompt{}

	if v, ok := m.current.(resourcetypetable.Model); ok {
		for _, rt := range v.SelectedResourceTypes {
			prompts = append(prompts, artifactPrompt{
				artifactDefinitionName: rt.Name,
			})
		}
	}

	return prompts
}

func buildArtifactTable(m Model, artdefName string) resourcetable.Model {
	creds, _ := m.listCredentials(artdefName)
	table := resourcetable.New(creds)
	return table
}
