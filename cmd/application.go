package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/commands/package/configure"
	"github.com/massdriver-cloud/mass/pkg/commands/package/patch"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/spf13/cobra"
)

var (
	appParamsPath   = "./params.json"
	appPatchQueries []string
)

func NewCmdApp() *cobra.Command {
	appCmd := &cobra.Command{
		Use:     "application",
		Aliases: []string{"app"},
		Short:   "Manage applications",
		Long:    mustRenderHelpDoc("application"),
	}

	appDeployCmd := &cobra.Command{
		Use:   `deploy <project>-<target>-<manifest>`,
		Short: "Deploy applications",
		Long:  mustRenderHelpDoc("application/deploy"),
		Args:  cobra.ExactArgs(1),
		RunE:  runAppDeploy,
	}

	appConfigureCmd := &cobra.Command{
		Use:     `configure <project>-<target>-<manifest>`,
		Short:   "Configure application",
		Aliases: []string{"cfg"},
		Long:    mustRenderHelpDoc("application/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runAppConfigure,
	}

	appConfigureCmd.Flags().StringVarP(&appParamsPath, "params", "p", appParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	appPatchCmd := &cobra.Command{
		Use:     `patch <project>-<target>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Aliases: []string{"cfg"},
		Long:    mustRenderHelpDoc("application/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runAppPatch,
	}

	appPatchCmd.Flags().StringArrayVarP(&appPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appConfigureCmd)
	appCmd.AddCommand(appPatchCmd)

	return appCmd
}

func runAppDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	_, err := commands.DeployPackage(client, config.OrgID, name)

	return err
}

func runAppConfigure(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)
	params := map[string]interface{}{}
	if err := files.Read(appParamsPath, &params); err != nil {
		return err
	}

	_, err := configure.Run(client, config.OrgID, packageSlugOrID, params)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Configuring: %s", name)
	fmt.Println(msg)

	return err
}

func runAppPatch(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	_, err := patch.Run(client, config.OrgID, packageSlugOrID, appPatchQueries)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Patching: %s", name)
	fmt.Println(msg)

	return err
}
