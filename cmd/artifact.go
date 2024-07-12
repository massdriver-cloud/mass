package cmd

import (
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/artifact"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/spf13/cobra"
)

func NewCmdArtifact() *cobra.Command {
	artifactCmd := &cobra.Command{
		Use:   "artifact",
		Short: "Manage artifacts",
		Long:  helpdocs.MustRender("artifact"),
	}

	// Import
	artifactImportCmd := &cobra.Command{
		Use:   `import`,
		Short: "Import a custom artifact",
		Long:  helpdocs.MustRender("artifact/import"),
		RunE:  runArtifactImport,
	}
	artifactImportCmd.Flags().StringP("name", "n", "", "Artifact name")
	artifactImportCmd.Flags().StringP("type", "t", "", "Artifact type")
	artifactImportCmd.Flags().StringP("file", "f", "", "Artifact file")

	artifactCmd.AddCommand(artifactImportCmd)

	return artifactCmd
}

func runArtifactImport(cmd *cobra.Command, args []string) error {
	artifactName, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	artifactType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	artifactFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	c, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	gqlclient := api.NewClient(c.URL, c.APIKey)

	promptData := artifact.ImportedArtifact{Name: artifactName, Type: artifactType, File: artifactFile}
	promptErr := artifact.RunArtifactImportPrompt(gqlclient, c.OrgID, &promptData)
	if promptErr != nil {
		return promptErr
	}

	_, importErr := commands.ArtifactImport(gqlclient, c.OrgID, promptData.Name, promptData.Type, promptData.File)
	return importErr
}
