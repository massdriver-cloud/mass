package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// This has 4 spaces at the beginning to make it look nice in md. It
// turns it into a code block which preserves spaces/returns
var rootCmdHelp = `
    ███    ███  █████  ███████ ███████
    ████  ████ ██   ██ ██      ██
    ██ ████ ██ ███████ ███████ ███████
    ██  ██  ██ ██   ██      ██      ██
    ██      ██ ██   ██ ███████ ███████

Massdriver Cloud CLI

Develop and publish private bundles.

Configure and deploying infrastructure and applications.
`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "mass",
	Short:             "Massdriver Cloud CLI",
	Long:              rootCmdHelp,
	DisableAutoGenTag: true,
}

// attributesToMap converts cobra's StringToString flag value into the
// map[string]any shape the API inputs expect. Returns nil when no entries are
// present so the field marshals to JSON null and the server treats it as
// absent rather than rejecting an empty Map.
func attributesToMap(attrs map[string]string) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

// stringMapToAny preserves an existing attribute map (read off a Project,
// Environment, or Component) when an update command needs to round-trip it
// without modification.
func stringMapToAny(m map[string]string) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(NewCmdBundle())
	rootCmd.AddCommand(NewCmdComponent())
	rootCmd.AddCommand(NewCmdDeployment())
	rootCmd.AddCommand(NewCmdDocs())
	rootCmd.AddCommand(NewCmdEnvironment())
	rootCmd.AddCommand(NewCmdInstance())
	rootCmd.AddCommand(NewCmdProject())
	rootCmd.AddCommand(NewCmdResource())
	rootCmd.AddCommand(NewCmdType())
	rootCmd.AddCommand(NewCmdSchema())
	rootCmd.AddCommand(NewCmdServer())
	rootCmd.AddCommand(NewCmdVersion())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
