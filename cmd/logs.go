package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs [deployment-id]",
		Short: "Get deployment logs",
		Long:  helpdocs.MustRender("logs"),
		Args:  cobra.ExactArgs(1),
		RunE:  runLogs,
		Example: `  # Get logs for a deployment
  mass logs 12345678-1234-1234-1234-123456789012`,
	}

	return logsCmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	deploymentID := args[0]
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	logs, err := api.GetDeploymentLogs(ctx, mdClient, deploymentID)
	if err != nil {
		return fmt.Errorf("error getting deployment logs: %w", err)
	}

	// Output logs to stdout
	for _, log := range logs {
		fmt.Fprint(os.Stdout, log.Content)
	}

	return nil
}
