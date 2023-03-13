// TODO: commands/previewenv/initialize/*.go
package initializeprevenv

import (
	"io"
	"log"

	"github.com/Khan/genqlient/graphql"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/debuglog"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
)

func Run(client graphql.Client, orgID string, projectSlug string, stdin io.Reader, stdout io.Writer) (*tea.Program, error) {
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("projectSlug", projectSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	cmdLog.Info().Str("id", project.ID).Msg("Found project.")

	artDefTable := artdeftable.New(api.ListCredentialTypes())
	m := New(artDefTable)

	m.ListCredentials = func(artDefType string) ([]*api.Artifact, error) {
		return api.ListCredentials(client, orgID, artDefType)
	}

	// TODO: can we return the m and let cobra start the command, we would need no
	// stdin / stdout mocks...
	// should all params be fields on the model?
	p := tea.NewProgram(m)
	result, err := p.Run()
	_ = err

	updatedModel, _ := (result).(Model)

	// TODO Get the data out to file...
	for _, p := range updatedModel.prompts {
		log.Printf("Artifact %v\n", p.selection)
	}

	return p, nil
}
