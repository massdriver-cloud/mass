package initialize

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/charmbracelet/bubbles/key"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/debuglog"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
)

// New returns a new tea.Model for initializing a preview env config
func New(client graphql.Client, orgID string, projectSlug string) (*Model, error) {
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("projectSlug", projectSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	cmdLog.Info().Str("id", project.ID).Msg("Found project.")

	artDefTable := artdeftable.New(api.ListCredentialTypes())

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

	m := Model{
		project:      project,
		keys:         keys,
		promptCursor: -1,
		artDefTable:  artDefTable,
		listCredentials: func(artDefType string) ([]*api.Artifact, error) {
			return api.ListCredentials(client, orgID, artDefType)
		},
	}

	return &m, nil
}
