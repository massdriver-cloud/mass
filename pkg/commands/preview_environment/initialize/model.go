package initialize

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/tui/components/artdeftable"
	"github.com/massdriver-cloud/mass/pkg/tui/components/artifacttable"
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

type Model struct {
	keys            KeyMap
	help            help.Model
	quitting        bool
	loaded          bool
	listCredentials func(string) ([]*api.Artifact, error)
	project         *api.Project
	environment     *api.GetEnvironmentByIdEnvironment

	// ui mode
	mode mode

	// initial artdef selector
	artDefTable tea.Model

	// current model
	current      tea.Model
	prompts      []artifactPrompt
	promptCursor int
}

func (m Model) PreviewConfig() *api.PreviewConfig {
	var credentials []api.Credential

	for _, p := range m.prompts {
		credentials = append(credentials, api.Credential{ArtifactDefinitionType: p.artifactDefinitionName, ArtifactId: p.selection.ID})
	}

	return &api.PreviewConfig{
		Packages:    m.hydratePackages(),
		Credentials: credentials,
		ProjectSlug: m.project.Slug,
	}
}

// hydratePackages uses the reference environment to build the preview package JSON and if no params
// are set in the env use the default values from the project
func (m Model) hydratePackages() map[string]api.PreviewPackage {
	packages := make(map[string]api.PreviewPackage)

	defaults := m.project.GetDefaultParams()

	for _, p := range m.environment.Packages {
		previwPackage := graphqlPackagetoPreviewPackage(p)
		if defaultPackage, ok := defaults[p.Manifest.Slug]; ok {
			// Check if we have params already, if not then use the default from the project
			// This does not fill out all available fields, only ones set at the project level
			if len(previwPackage.Params) == 0 {
				previwPackage.Params = defaultPackage.Params
			}
		}
		packages[p.Manifest.Slug] = previwPackage
	}

	return packages
}

func graphqlPackagetoPreviewPackage(in api.GetEnvironmentByIdEnvironmentPackagesPackage) api.PreviewPackage {
	var transformedPackage api.PreviewPackage

	transformedPackage.Params = in.Params

	for _, remote := range in.GetRemoteReferences() {
		r := api.RemoteRef{
			ArtifactID: remote.Artifact.Id,
			Field:      remote.Field,
		}
		transformedPackage.RemoteReferences = append(transformedPackage.RemoteReferences, r)
	}

	for _, secret := range in.GetSecretFields() {
		s := api.Secret{
			Name:     secret.Name,
			Required: secret.Required,
		}
		transformedPackage.Secrets = append(transformedPackage.Secrets, s)
	}

	return transformedPackage
}

func (m Model) Init() tea.Cmd {
	return nil
}

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
				selectedArtifacts := m.current.(artifacttable.Model).SelectedArtifacts
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

	if v, ok := m.current.(artdeftable.Model); ok {
		for _, artdef := range v.SelectedArtifactDefinitions {
			prompts = append(prompts, artifactPrompt{
				artifactDefinitionName: artdef.Name,
			})
		}
	}

	return prompts
}

func buildArtifactTable(m Model, artdefName string) artifacttable.Model {
	creds, _ := m.listCredentials(artdefName)
	table := artifacttable.New(creds)
	return table
}
