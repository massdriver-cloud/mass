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

	credentials, err := api.ListCredentials(ctx, mdClient)
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	tbl := cli.NewTable("ID", "Type", "Name", "Updated At")

	for _, credential := range credentials {
		name := credential.Name
		if len(name) > 60 {
			name = name[:60] + "..."
		}
		updatedAt := credential.UpdatedAt.Format("2006-01-02 15:04:05")
		tbl.AddRow(credential.ID, credential.Type, name, updatedAt)
	}

	tbl.Print()

	return nil
}
