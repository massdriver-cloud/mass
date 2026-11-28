package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	cmd.Flags().Duration("stash", dashboardStash, "How often to read the stash when it cannot be streamed, as when the profile signs in without a personal access token or while a stream reconnects; 0 reads it with each refresh")
	cmd.Flags().String("env", "", "Environment to center on, by id; the most recently deployed one when left out")
	cmd.Flags().Int("depth", dashboard.DefaultDepth, "How many hops from the focus to read")
	cmd.Flags().String("theme", dashboard.DefaultTheme, "How the map is drawn: "+strings.Join(dashboard.Themes(), ", "))
	cmd.Flags().String("render", dashboard.DefaultRender, "How the map's dots are drawn: braille dots, or solid blocks for terminals that draw block characters themselves, such as Ghostty")
	cmd.Flags().String("palette", dashboard.DefaultPalette, "Colors and backdrop: "+strings.Join(dashboard.Palettes(), ", "))
	cmd.Flags().Bool("watch", false, "Jump to any alarm that starts firing in view while the dashboard is open, and send a desktop notification")
	cmd.Flags().Bool("read-only", false, "Turn off approving and deploying from the map")
	cmd.Flags().String("playback", "", "Replay an incident recording written with w, offline")
	return cmd
}

const (
	dashboardRefresh = 30 * time.Second
	dashboardEvents  = 5 * time.Second
	dashboardStash   = 3 * time.Second
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
	stash, err := cmd.Flags().GetDuration("stash")
	if err != nil {
		return err
	}
	theme, err := cmd.Flags().GetString("theme")
	if err != nil {
		return err
	}
	render, err := cmd.Flags().GetString("render")
	if err != nil {
		return err
	}
	palette, err := cmd.Flags().GetString("palette")
	if err != nil {
		return err
	}
	focus, err := cmd.Flags().GetString("env")
	if err != nil {
		return err
	}
	depth, err := cmd.Flags().GetInt("depth")
	if err != nil {
		return err
	}
	watch, err := cmd.Flags().GetBool("watch")
	if err != nil {
		return err
	}
	readOnly, err := cmd.Flags().GetBool("read-only")
	if err != nil {
		return err
	}
	playback, err := cmd.Flags().GetString("playback")
	if err != nil {
		return err
	}
	ctx := context.Background()
	if playback != "" {
		return playRecording(ctx, playback, dashboard.Options{Config: dashboard.Config{Theme: theme, Render: render, ReadOnly: true}, Palette: palette})
	}
	mdClient, err := newMassdriverClient(cmd)
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}
	return dashboard.Run(ctx, dashboard.NewAPI(mdClient), dashboard.Options{Config: dashboard.Config{
		Org:       mdClient.Config().OrganizationID,
		Refresh:   refresh,
		EventPoll: events,
		StashPoll: stash,
		Watch:     watch,
		ReadOnly:  readOnly,
		Links:     dashboard.NewLinks(mdClient.URLs.Helper(ctx)),
		Theme:     theme,
		Render:    render,
		Focus:     focus,
		Depth:     depth,
	}, Palette: palette})
}

func playRecording(ctx context.Context, path string, opts dashboard.Options) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	rec, err := dashboard.LoadRecording(f)
	if err != nil {
		return err
	}
	opts.Config = rec.PlaybackConfig(opts.Config)
	return dashboard.Run(ctx, rec.API(), opts)
}
