package cmd

import (
	"github.com/massdriver-cloud/mass/docs/helpdocs"
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
		RunE:    runPkgConfigure,
	}

	appConfigureCmd.Flags().StringVarP(&appParamsPath, "params", "p", appParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	appDeployCmd := &cobra.Command{
		Use:   `deploy <project>-<env>-<manifest>`,
		Short: "Deploy applications",
		Long:  helpdocs.MustRender("application/deploy"),
		Args:  cobra.ExactArgs(1),
		RunE:  runPkgDeploy,
	}

	appDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	appPatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Aliases: []string{"cfg"},
		Long:    helpdocs.MustRender("application/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgPatch,
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
