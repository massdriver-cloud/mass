package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mass",
	Short: "Massdriver Cloud CLI",
	Long: `
	███    ███  █████  ███████ ███████
	████  ████ ██   ██ ██      ██
	██ ████ ██ ███████ ███████ ███████
	██  ██  ██ ██   ██      ██      ██
	██      ██ ██   ██ ███████ ███████

	Massdriver Cloud CLI

	Develop and publish private bundles.

	Configure and deploying infrastructure and applications.
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
