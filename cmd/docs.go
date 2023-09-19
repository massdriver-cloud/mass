package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Gen docs",
		Long:  "Gen docs",
		Run: func(cmd *cobra.Command, args []string) {
			dir, err := cmd.Flags().GetString("directory")
			if err != nil {
				log.Fatal(err)
			}
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			err = doc.GenMarkdownTree(rootCmd, dir)
			if err != nil {
				log.Fatal(err)
			}
		},
		Args:   cobra.NoArgs,
		Hidden: false,
	}

	cmd.Flags().StringP("directory", "d", "docs/generated", "directory to generate docs into")
	cmd.Flags().String("log-level", "info", "Set the log level for the server. Options are [debug, info, warn, error]")

	return cmd
}
