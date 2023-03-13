// TODO: rename file from 'main.go' unless this just holds New()
package initialize

import (
	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/debuglog"
	"github.com/massdriver-cloud/mass/internal/tui/components/artdeftable"
)

func Run(client graphql.Client, orgID string, projectSlug string) (*Model, error) {
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

	return &m, nil
}
