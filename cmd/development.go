package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	commands "github.com/massdriver-cloud/mass/internal/commands/development"
	"github.com/spf13/cobra"
)

var developmentSimulatorHelp = helpdocs.MustRender("development/simulator")

var developmentCmd = &cobra.Command{
	Use:   "development",
	Short: "dev",
}

var developmentSimulatorCmd = &cobra.Command{
	Use:   "simulator",
	Short: "sim",
	Long:  developmentSimulatorHelp,
	RunE:  runDevelopmentSimulator,
}

func init() {
	rootCmd.AddCommand(developmentCmd)
	developmentCmd.AddCommand(developmentSimulatorCmd)
}

func runDevelopmentSimulator(cmd *cobra.Command, args []string) error {
	fmt.Println(args)
	return commands.StartDevelopmentSimulator()
}
