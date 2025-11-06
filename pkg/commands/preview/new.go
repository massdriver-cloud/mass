package preview

import (
	"context"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/debuglog"
	"github.com/massdriver-cloud/mass/pkg/tui/components/artdeftable"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type KeyMap struct {
	Quit key.Binding
	Next key.Binding
	Back key.Binding
	Help key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Next, k.Quit},
		{k.Help},
	}
}

// RunNew returns a new tea.Model for initializing a preview env config
func RunNew(ctx context.Context, mdClient *client.Client, projectSlug string) (*Model, error) {
	cmdLog := debuglog.Log().With().Str("orgID", mdClient.Config.OrganizationID).Str("projectSlug", projectSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	project, err := api.GetProject(ctx, mdClient, projectSlug)

	if err != nil {
		return nil, err
	}

	cmdLog.Info().Str("id", project.ID).Msg("Found project.")

	artDefTable := artdeftable.New(api.ListCredentialTypes(ctx, mdClient))

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
		keys:         keys,
		promptCursor: -1,
		artDefTable:  artDefTable,
		listCredentials: func(artDefType string) ([]*api.Artifact, error) {
			return api.ListArtifactsByType(ctx, mdClient, artDefType)
		},
	}

	m.help.ShowAll = true

	return &m, nil
}
