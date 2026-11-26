package cmd

import (
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/commands/credentials"
	"github.com/spf13/cobra"
)

// NewCmdCredentials returns a cobra command that prints credentials other tools
// use to reach Massdriver.
func NewCmdCredentials() *cobra.Command {
	credentialsCmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"creds"},
		Short:   "Print credentials for other tools to reach Massdriver",
		Long:    helpdocs.MustRender("credentials"),
	}

	s3Cmd := &cobra.Command{
		Use:     "s3",
		Short:   "Print AWS credentials for the Massdriver S3 API",
		Example: `mass credentials s3 --output env`,
		Long:    helpdocs.MustRender("credentials/s3"),
		Args:    cobra.NoArgs,
		RunE:    runCredentialsS3,
	}
	s3Cmd.Flags().StringP("output", "o", "process", "Output format (process, env)")

	credentialsCmd.AddCommand(s3Cmd)

	return credentialsCmd
}

func runCredentialsS3(cmd *cobra.Command, args []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	if err = credentials.ValidateS3Output(output); err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, err := newMassdriverClient(cmd)
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return credentials.RunS3(cmd.Context(), credentials.NewAPI(mdClient), output, os.Stdout)
}
