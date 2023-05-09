package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/commands/package/configure"
	"github.com/massdriver-cloud/mass/internal/commands/package/patch"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/spf13/cobra"
)

var infraParamsPath = "./params.json"
var infraPatchQueries []string
var infraCmdHelp = mustRenderHelpDoc("infrastructure")
var infraDeployCmdHelp = mustRenderHelpDoc("infrastructure/deploy")
var infraPatchCmdHelp = mustRenderHelpDoc("infrastructure/patch")
var infraConfigureCmdHelp = mustRenderHelpDoc("infrastructure/configure")

var infraCmd = &cobra.Command{
	Use:     "infrastructure",
	Aliases: []string{"infra"},
	Short:   "Manage infrastructure.",
	Long:    infraCmdHelp,
}

var infraDeployCmd = &cobra.Command{
	Use:   `deploy <project>-<target>-<manifest>`,
	Short: "Deploy infrastructure",
	Long:  infraDeployCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runInfraDeploy,
}

var infraConfigureCmd = &cobra.Command{
	Use:     `configure <project>-<target>-<manifest>`,
	Short:   "Configure infrastructure",
	Aliases: []string{"cfg"},
	Long:    infraConfigureCmdHelp,
	Args:    cobra.ExactArgs(1),
	RunE:    runInfraConfigure,
}

var infraPatchCmd = &cobra.Command{
	Use:     `patch <project>-<target>-<manifest>`,
	Short:   "Patch individual package parameter values",
	Aliases: []string{"cfg"},
	Long:    infraPatchCmdHelp,
	Args:    cobra.ExactArgs(1),
	RunE:    runInfraPatch,
}

func init() {
	rootCmd.AddCommand(infraCmd)
	infraCmd.AddCommand(infraDeployCmd)
	infraCmd.AddCommand(infraConfigureCmd)
	infraCmd.AddCommand(infraPatchCmd)

	infraConfigureCmd.Flags().StringVarP(&infraParamsPath, "params", "p", infraParamsPath, "Path to params JSON file. This file supports bash interpolation.")
	infraPatchCmd.Flags().StringArrayVarP(&infraPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")
}

func runInfraDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	_, err := commands.DeployPackage(client, config.OrgID, name)

	return err
}

func runInfraConfigure(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)
	params := map[string]interface{}{}
	if err := files.Read(infraParamsPath, &params); err != nil {
		return err
	}

	_, err := configure.Run(client, config.OrgID, packageSlugOrID, params)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Configuring: %s", name)
	fmt.Println(msg)

	return err
}

func runInfraPatch(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	_, err := patch.Run(client, config.OrgID, packageSlugOrID, infraPatchQueries)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Patching: %s", name)
	fmt.Println(msg)

	return err
}
