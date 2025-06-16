package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdApp() *cobra.Command {
	appCmd := &cobra.Command{
		Use:        "application",
		Aliases:    []string{"app"},
		Deprecated: "This has been renamed to `package`. This command will be removed in v2.",
	}

	appConfigureCmd := &cobra.Command{
		Use:        `configure <project>-<env>-<manifest>`,
		Aliases:    []string{"cfg"},
		Deprecated: "This has been moved under `package`. This command will be removed in v2.",
		Args:       cobra.ExactArgs(1),
		RunE:       runPkgConfigure,
	}

	appConfigureCmd.Flags().StringVarP(&pkgParamsPath, "params", "p", pkgParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	appDeployCmd := &cobra.Command{
		Use:        `deploy <project>-<env>-<manifest>`,
		Deprecated: "This has been moved under `package`. This command will be removed in v2.",
		Args:       cobra.ExactArgs(1),
		RunE:       runPkgDeploy,
	}

	appDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	appPatchCmd := &cobra.Command{
		Use:        `patch <project>-<env>-<manifest>`,
		Deprecated: "This has been moved under `package`. This command will be removed in v2.",
		Aliases:    []string{"cfg"},
		Args:       cobra.ExactArgs(1),
		RunE:       runPkgPatch,
	}

	appPatchCmd.Flags().StringArrayVarP(&pkgPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	// app and infra are the same, lets reuse a get command/template here.
	pkgGetCmd := &cobra.Command{
		Use:        `get  <project>-<env>-<manifest>`,
		Deprecated: "This has been moved under `package`. This command will be removed in v2.",
		Aliases:    []string{"g"},
		Args:       cobra.ExactArgs(1), // Enforce exactly one argument
		RunE:       runPkgGet,
	}

	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appConfigureCmd)
	appCmd.AddCommand(appPatchCmd)
	appCmd.AddCommand(pkgGetCmd)

	return appCmd
}
