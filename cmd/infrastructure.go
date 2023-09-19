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
	infraParamsPath   = "./params.json"
	infraPatchQueries []string
)

func NewCmdInfra() *cobra.Command {
	infraCmd := &cobra.Command{
		Use:     "infrastructure",
		Aliases: []string{"infra"},
		Short:   "Manage infrastructure",
		Long:    helpdocs.MustRender("infrastructure"),
	}

	infraConfigureCmd := &cobra.Command{
		Use:     `configure <project>-<target>-<manifest>`,
		Short:   "Configure infrastructure",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("infrastructure/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInfraConfigure,
	}
	infraConfigureCmd.Flags().StringVarP(&infraParamsPath, "params", "p", infraParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	infraDeployCmd := &cobra.Command{
		Use:   `deploy <project>-<target>-<manifest>`,
		Short: "Deploy infrastructure",
		Long:  helpdocs.MustRender("infrastructure/deploy"),
		Args:  cobra.ExactArgs(1),
		RunE:  runInfraDeploy,
	}

	infraPatchCmd := &cobra.Command{
		Use:     `patch <project>-<target>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("infrastructure/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInfraPatch,
	}
	infraPatchCmd.Flags().StringArrayVarP(&infraPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	infraCmd.AddCommand(infraConfigureCmd)
	infraCmd.AddCommand(infraDeployCmd)
	infraCmd.AddCommand(infraPatchCmd)

	return infraCmd
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
