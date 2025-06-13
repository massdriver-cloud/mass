package cmd

import (
	"github.com/massdriver-cloud/mass/docs/helpdocs"
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
		Use:     `configure <project>-<env>-<manifest>`,
		Short:   "Configure infrastructure",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("infrastructure/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgConfigure,
	}

	infraConfigureCmd.Flags().StringVarP(&infraParamsPath, "params", "p", infraParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	infraDeployCmd := &cobra.Command{
		Use:   `deploy <project>-<env>-<manifest>`,
		Short: "Deploy infrastructure",
		Long:  helpdocs.MustRender("infrastructure/deploy"),
		Args:  cobra.ExactArgs(1),
		RunE:  runPkgDeploy,
	}

	infraDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	infraPatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("infrastructure/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgPatch,
	}

	infraPatchCmd.Flags().StringArrayVarP(&infraPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	// app and infra are the same, lets reuse a get command/template here.
	pkgGetCmd := &cobra.Command{
		Use:     `get <project>-<env>-<manifest>`,
		Short:   "Get an infrastructure package",
		Aliases: []string{"g"},
		Long:    helpdocs.MustRender("package/get"),
		Args:    cobra.ExactArgs(1), // Enforce exactly one argument
		RunE:    runPkgGet,
	}

	infraCmd.AddCommand(infraConfigureCmd)
	infraCmd.AddCommand(infraDeployCmd)
	infraCmd.AddCommand(infraPatchCmd)
	infraCmd.AddCommand(pkgGetCmd)

	return infraCmd
}
