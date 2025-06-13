package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
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
		Long:    helpdocs.MustRender("application"),
	}

	appConfigureCmd := &cobra.Command{
		Use:     `configure <project>-<env>-<manifest>`,
		Short:   "Configure application",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("application/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runAppConfigure,
	}

	appConfigureCmd.Flags().StringVarP(&appParamsPath, "params", "p", appParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	appDeployCmd := &cobra.Command{
		Use:   `deploy <project>-<env>-<manifest>`,
		Short: "Deploy applications",
		Long:  helpdocs.MustRender("application/deploy"),
		Args:  cobra.ExactArgs(1),
		RunE:  runAppDeploy,
	}

	appDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	appPatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("application/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runAppPatch,
	}

	appPatchCmd.Flags().StringArrayVarP(&appPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	// app and infra are the same, lets reuse a get command/template here.
	pkgGetCmd := &cobra.Command{
		Use:     `get  <project>-<env>-<manifest>`,
		Short:   "Get an applicaton package",
		Aliases: []string{"g"},
		Long:    helpdocs.MustRender("package/get"),
		Args:    cobra.ExactArgs(1), // Enforce exactly one argument
		RunE:    runPkgGet,
	}

	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appConfigureCmd)
	appCmd.AddCommand(appPatchCmd)
	appCmd.AddCommand(pkgGetCmd)

	return appCmd
}

func runAppDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}

	_, err = commands.DeployPackage(client, config.OrgID, name, msg)

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
