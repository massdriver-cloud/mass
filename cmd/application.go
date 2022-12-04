package cmd

import "github.com/spf13/cobra"

const appCmdHelp = `
Configure and deploy applications managed with Massdriver.
`

var appCmd = &cobra.Command{
	Use:     "application",
	Aliases: []string{"app"},
	Short:   "Configure & deploy applications.",
	Long:    appCmdHelp,
}

func init() {
	rootCmd.AddCommand(appCmd)
}
