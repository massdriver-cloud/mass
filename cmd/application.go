package cmd

import (
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/cobra"
)

var appCmdHelp = mustRenderHelpDoc("app")

var appDeployCmdHelp = mustRenderHelpDoc("app/deploy")

var appCmd = &cobra.Command{
	Use:     "application",
	Aliases: []string{"app"},
	Short:   "Configure & deploy applications.",
	Long:    appCmdHelp,
}

var appDeployCmd = &cobra.Command{
	Use:   `deploy <project>-<target>-<manifest>`,
	Short: "Deploy an application on Massdriver",
	Long:  appDeployCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runAppDeploy,
}

func init() {
	rootCmd.AddCommand(appCmd)
	appCmd.AddCommand(appDeployCmd)
}

func runAppDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	_, err := commands.DeployPackage(client, c.OrgID, name)

	return err
}
