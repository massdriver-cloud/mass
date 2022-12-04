package cmd

import "github.com/spf13/cobra"

const infraCmdHelp = `
Configure and deploy infrastructure managed with Massdriver.
`

var infraCmd = &cobra.Command{
	Use:     "infrastructure",
	Aliases: []string{"infra"},
	Short:   "Configure & deploy infrastructure.",
	Long:    infraCmdHelp,
}

func init() {
	rootCmd.AddCommand(infraCmd)
}
