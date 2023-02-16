package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

//go:embed helpdocs/*.md helpdocs/**
var helpdocs embed.FS

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
	Use:   "mass",
	Short: "Massdriver Cloud CLI",
	Long:  rootCmdHelp,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func mustRenderHelpDoc(name string) string {
	path := fmt.Sprintf("helpdocs/%s.md", name)
	data, err := helpdocs.ReadFile(path)
	if err != nil {
		panic(err)
	}

	out, err := glamour.Render(string(data), "auto")
	if err != nil {
		panic(err)
	}
	return out
}
