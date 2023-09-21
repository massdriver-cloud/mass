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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(NewCmdApp())
	rootCmd.AddCommand(NewCmdArtifact())
	rootCmd.AddCommand(NewCmdBundle())
	rootCmd.AddCommand(NewCmdDocs())
	rootCmd.AddCommand(NewCmdImage())
	rootCmd.AddCommand(NewCmdInfra())
	rootCmd.AddCommand(NewCmdPreview())
	rootCmd.AddCommand(NewCmdSchema())
	rootCmd.AddCommand(NewCmdServer())
	rootCmd.AddCommand(NewCmdVersion())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
