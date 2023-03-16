package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/commands/package/configure"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/spf13/cobra"
)

var appParamsPath = "./params.json"

var appCmdHelp = mustRenderHelpDoc("application")
var appDeployCmdHelp = mustRenderHelpDoc("application/deploy")
var appConfigureCmdHelp = mustRenderHelpDoc("application/configure")

var appCmd = &cobra.Command{
	Use:     "application",
	Aliases: []string{"app"},
	Short:   "Manage applications.",
	Long:    appCmdHelp,
}

var appDeployCmd = &cobra.Command{
	Use:   `deploy <project>-<target>-<manifest>`,
	Short: "Deploy applications",
	Long:  appDeployCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runAppDeploy,
}

var appConfigureCmd = &cobra.Command{
	Use:     `configure <project>-<target>-<manifest>`,
	Short:   "Configure application",
	Aliases: []string{"cfg"},
	Long:    appConfigureCmdHelp,
	Args:    cobra.ExactArgs(1),
	RunE:    runAppConfigure,
}

func init() {
	rootCmd.AddCommand(appCmd)
	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appConfigureCmd)

	appConfigureCmd.Flags().StringVarP(&appParamsPath, "params", "p", appParamsPath, "Path to params JSON file. This file supports bash interpolation.")
}

func runAppDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	_, err := commands.DeployPackage(client, c.OrgID, name)

	return err
}

func runAppConfigure(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)
	params := map[string]interface{}{}
	if err := files.Read(appParamsPath, &params); err != nil {
		return err
	}

	_, err := configure.Run(client, c.OrgID, packageSlugOrID, params)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Configuring: %s", name)
	fmt.Println(msg)

	return err
}

// TODO: patch apps
