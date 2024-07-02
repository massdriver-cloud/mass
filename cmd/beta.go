package cmd

import (
	"github.com/massdriver-cloud/mass/cmd/beta"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/spf13/cobra"
)

func NewCmdBeta() *cobra.Command {
	betaCmd := &cobra.Command{
		Use:   "beta",
		Short: "Beta features and capabilities",
		Long:  helpdocs.MustRender("beta"),
	}

	betaCmd.AddCommand(beta.NewCmdApply())
	betaCmd.AddCommand(beta.NewCmdDestroy())

	return betaCmd
}
