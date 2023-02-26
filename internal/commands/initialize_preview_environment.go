package commands

import (
	"fmt"
	"io"
	"log"

	"github.com/Khan/genqlient/graphql"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/debuglog"
	"github.com/massdriver-cloud/mass/internal/tui/selectable"
)

func InitializePreviewEnvironment(client graphql.Client, orgID string, projectSlug string, stdin io.Reader, stdout io.Writer) (*PreviewConfig, error) {
	cmdLog := debuglog.Log().With().Str("orgID", orgID).Str("projectSlug", projectSlug).Logger()
	cmdLog.Info().Msg("Initializing preview environment.")
	project, err := api.GetProject(client, orgID, projectSlug)

	if err != nil {
		return nil, err
	}

	nameColumn := "credentialType"
	columns := []table.Column{
		table.NewColumn(nameColumn, "Credential Type", 40),
	}

	rows := []table.Row{}
	credentialTypes := api.ListCredentialTypes()
	for i := range credentialTypes {
		credentialType := credentialTypes[i]
		// TODO: set the type in the table as metadta and present a pretty name
		row := table.NewRow(table.RowData{nameColumn: credentialType.Name})
		rows = append(rows, row)
	}

	model := selectable.
		New(columns).
		WithTitle("Which credential types do you want to use?").
		WithRows(rows).
		Focused(true).
		WithMinimum(1)

	P = tea.NewProgram(
		model, // Note: can be given multiple models... and use bubble tea to swap between them based on input
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
	)

	// TODO: Should commands return a list of models so all cmds can start the program w/o passing stdin/stdout?
	result, err := P.Run()

	if err != nil {
		cmdLog.Error().Err(err)
		log.Fatal(err)
	}

	updatedModel, _ := (result).(selectable.Model)

	credentials := map[string]string{}

	for _, row := range updatedModel.SelectedRows() {
		credentialType := (row[nameColumn]).(string)
		model = newArtifactSelectionModel(client, orgID, credentialType)

		P = tea.NewProgram(
			model,
			tea.WithInput(stdin),
			tea.WithOutput(stdout),
		)

		result, err := P.Run()

		if err != nil {
			cmdLog.Error().Err(err)
			log.Fatal(err)
		}

		updatedModel, _ := (result).(selectable.Model)
		selectedRows := updatedModel.SelectedRows()
		// only one row is allowed to be selected, get it and stick in in the map
		firstRow := selectedRows[0]

		if firstRow != nil {
			credentials[credentialType] = (firstRow["id"]).(string)
		}
	}

	cfg := PreviewConfig{
		PackageParams: project.DefaultParams,
		Credentials:   credentials,
	}

	return &cfg, nil
}

func newArtifactSelectionModel(client graphql.Client, orgID string, artifactType string) selectable.Model {
	artifactList, _ := api.ListCredentials(client, orgID, artifactType)

	nameColumn := "name"
	idColumn := "id"

	columns := []table.Column{
		table.NewColumn(nameColumn, "Name", 40),
		table.NewColumn(idColumn, "Artifact ID", 40),
	}

	rows := []table.Row{}
	for _, artifact := range artifactList {
		row := table.NewRow(table.RowData{nameColumn: artifact.Name, idColumn: artifact.ID})
		rows = append(rows, row)
	}

	model := selectable.
		New(columns).
		WithTitle(fmt.Sprintf("Select a credential for %s", artifactType)).
		WithRows(rows).
		Focused(true).
		WithMaximum(1)

	return model
}
