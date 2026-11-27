package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/spf13/cobra"
)

// NewCmdDashboard returns a cobra command that opens the live dashboard.
func NewCmdDashboard() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"dash", "constellation"},
		Short:   "Watch your environments, alarms, and runtimes live",
		Long:    helpdocs.MustRender("dashboard"),
		Args:    cobra.NoArgs,
		RunE:    runDashboard,
	}
	cmd.Flags().Duration("refresh", dashboardRefresh, "How often to re-read the organization; 0 reads once")
	cmd.Flags().Duration("events", dashboardEvents, "How often to read new activity from the audit log; 0 turns the live ticker off")
	cmd.Flags().Bool("watch", false, "Jump to any alarm that starts firing while the dashboard is open, and send a desktop notification")
	return cmd
}

const (
	dashboardRefresh = 30 * time.Second
	dashboardEvents  = 5 * time.Second
)

func runDashboard(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true
	refresh, err := cmd.Flags().GetDuration("refresh")
	if err != nil {
		return err
	}
	events, err := cmd.Flags().GetDuration("events")
	if err != nil {
		return err
	}
	watch, err := cmd.Flags().GetBool("watch")
	if err != nil {
		return err
	}
	mdClient, err := newMassdriverClient(cmd)
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}
	ctx := context.Background()
	return dashboard.Run(ctx, dashboard.NewAPI(mdClient), dashboard.Options{Config: dashboard.Config{
		Org:       mdClient.Config().OrganizationID,
		Refresh:   refresh,
		EventPoll: events,
		Watch:     watch,
		Links:     dashboard.NewLinks(mdClient.URLs.Helper(ctx)),
	}})
}
