package cmd

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

func NewCmdCredential() *cobra.Command {
	credentialCmd := &cobra.Command{
		Use:     "credential",
		Short:   "Credential management",
		Long:    helpdocs.MustRender("credential"),
		Aliases: []string{"cred"},
	}

	credentialListCmd := &cobra.Command{
		Use:   "list",
		Short: "List credentials",
		Long:  helpdocs.MustRender("credential/list"),
		RunE:  runCredentialList,
	}

	credentialCmd.AddCommand(credentialListCmd)

	return credentialCmd
}

func runCredentialList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	credentialTypes := api.ListCredentialTypes(ctx, mdClient)
	allCredentials := []*api.Artifact{}

	for _, credType := range credentialTypes {
		credentials, err := api.ListCredentials(ctx, mdClient, credType.Name)
		if err != nil {
			return err
		}
		allCredentials = append(allCredentials, credentials...)
	}

	tbl := cli.NewTable("ID", "Name", "Updated At")

	for _, credential := range allCredentials {
		tbl.AddRow(credential.ID, credential.Name, credential.UpdatedAt)
	}

	tbl.Print()

	return nil
}
