package cmd

import (
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var artifactCmdHelp = mustRenderHelpDoc("artifact")
var artifactImportCmdHelp = mustRenderHelpDoc("artifact/import")

var artifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage applications.",
	Long:  artifactCmdHelp,
}

// Import
var artifactImportCmd = &cobra.Command{
	Use:   `import -n <name> -t <type> -f <file>`,
	Short: "Import a custom artifact",
	Long:  artifactImportCmdHelp,
	RunE:  runArtifactImport,
}

func init() {
	rootCmd.AddCommand(artifactCmd)

	artifactCmd.AddCommand(artifactImportCmd)
	artifactImportCmd.Flags().StringP("name", "n", "", "Artifact name")
	artifactImportCmd.Flags().StringP("type", "t", "", "Artifact type")
	artifactImportCmd.Flags().StringP("file", "f", "", "Artifact file")
	_ = artifactImportCmd.MarkFlagRequired("name")
	_ = artifactImportCmd.MarkFlagRequired("type")
	_ = artifactImportCmd.MarkFlagRequired("file")
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

	var fs = afero.NewOsFs()

	c, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	gqlclient := api.NewClient(c.URL, c.APIKey)

	return commands.ArtifactImport(gqlclient, c.OrgID, fs, artifactName, artifactType, artifactFile)
}
