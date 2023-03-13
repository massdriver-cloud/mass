// TODO: commands/previewenv/initialize/*.go
package initializeprevenv

import (
	"fmt"
	"io"

	"github.com/Khan/genqlient/graphql"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/debuglog"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
	"github.com/massdriver-cloud/mass/internal/tui/components/artifacttable"
)

func Run(client graphql.Client, orgID string, projectSlug string, stdin io.Reader, stdout io.Writer) (*tea.Program, error) {
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("projectSlug", projectSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	cmdLog.Info().Str("id", project.ID).Msg("Found project.")

	m := New()
	m.screens = []tea.Model{
		artdeftable.New(api.ListCredentialTypes()),
	}

	// this should make testing easy...
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

	// TODO Get the data out ...
	// TODO: Max selected on artifact screen should be one ... :(
	for _, screen := range updatedModel.screens {
		if v, ok := screen.(artifacttable.Model); ok {
			for _, a := range v.SelectedArtifacts {
				fmt.Printf("Artifact: %v\n", a.Name)
			}
		}
	}

	return p, nil
}
