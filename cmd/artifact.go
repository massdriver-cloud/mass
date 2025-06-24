package cmd

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/artifact"
	artifactcmd "github.com/massdriver-cloud/mass/pkg/commands/artifact"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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
	ctx := context.Background()

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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	promptData := artifact.ImportedArtifact{Name: artifactName, Type: artifactType, File: artifactFile}
	promptErr := artifact.RunArtifactImportPrompt(ctx, mdClient, &promptData)
	if promptErr != nil {
		return promptErr
	}

	_, importErr := artifactcmd.RunImport(ctx, mdClient, promptData.Name, promptData.Type, promptData.File)
	return importErr
}
