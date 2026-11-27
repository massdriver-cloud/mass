package cmd

import (
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/spf13/cobra"
)

// NewCmdAlpha returns a cobra command that groups commands for features in alpha.
func NewCmdAlpha() *cobra.Command {
	alphaCmd := &cobra.Command{
		Use:   "alpha",
		Short: "Commands for features in alpha",
		Long:  helpdocs.MustRender("alpha"),
	}

	alphaCmd.AddCommand(newCmdAlphaStash())
	return alphaCmd
}
