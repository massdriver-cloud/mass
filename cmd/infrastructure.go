package cmd

import (
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/cobra"
)

const infraCmdHelp = `
# Configure and deploy infrastructure managed with Massdriver.
`

var infraDeployCmdHelp = mustRender(`
# Deploy infrastructure on Massdriver.

This infrastructure must be published as a [bundle](https://docs.massdriver.cloud/applications) to Massdriver first and be configured for a given environment (target).

Learn more in the [docs](https://docs.massdriver.cloud/bundles).
`)

var infraCmd = &cobra.Command{
	Use:     "infrastructure",
	Aliases: []string{"infra"},
	Short:   "Configure & deploy infrastructure.",
	Long:    infraCmdHelp,
}

var infraDeployCmd = &cobra.Command{
	Use:   `deploy <project>-<target>-<manifest>`,
	Short: "Deploy cloud infrastructure on Massdriver",
	Long:  infraDeployCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runInfraDeploy,
}

func init() {
	rootCmd.AddCommand(infraCmd)
	infraCmd.AddCommand(infraDeployCmd)
}

func runInfraDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	_, err := commands.DeployPackage(client, c.OrgID, name)

	return err
}
