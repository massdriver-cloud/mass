package initialize

import (
	"context"

	"github.com/Khan/genqlient/graphql"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/debuglog"
	"github.com/massdriver-cloud/mass/pkg/tui/components/artdeftable"
)

// New returns a new tea.Model for initializing a preview env config
func New(client graphql.Client, orgID string, envSlug string) (*Model, error) {
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("envSlug", envSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	env, err := api.GetEnvironmentById(context.Background(), client, orgID, envSlug)
	if err != nil {
		return nil, err
	}

	project, err := api.GetProject(client, orgID, env.Environment.Project.Id)
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
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}

	m := Model{
		help:         help.New(),
		project:      project,
		environment:  &env.Environment,
		keys:         keys,
		promptCursor: -1,
		artDefTable:  artDefTable,
		listCredentials: func(artDefType string) ([]*api.Artifact, error) {
			return api.ListCredentials(client, orgID, artDefType)
		},
	}

	m.help.ShowAll = true

	return &m, nil
}
