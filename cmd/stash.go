package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/commands/stash"
	"github.com/spf13/cobra"
)

// NewCmdStash returns a cobra command for stash entries.
func NewCmdStash() *cobra.Command {
	stashCmd := &cobra.Command{
		Use:   "stash",
		Short: "Store and read small values in a project, environment, component, instance, or deployment",
		Long:  helpdocs.MustRender("stash"),
	}

	getCmd := &cobra.Command{
		Use:   "get <scope>.<key>",
		Short: "Print a stash entry's value",
		Long:  helpdocs.MustRender("stash/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runStashGet,
	}
	getCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	setCmd := &cobra.Command{
		Use:   "set <scope>.<key> <value>",
		Short: "Set a stash entry's value",
		Long:  helpdocs.MustRender("stash/set"),
		Args:  cobra.ExactArgs(2),
		RunE:  runStashSet,
	}
	setCmd.Flags().String("format", "text", "How the value is read (text, json)")
	setCmd.Flags().Int("ttl", 0, "Delete the entry this many seconds after the write")
	setCmd.Flags().Bool("immutable", false, "Keep the first value; later writes fail")
	setCmd.Flags().Bool("sensitive", false, "Hide the value from callers that cannot write the scope")

	deleteCmd := &cobra.Command{
		Use:     "delete <scope>.<key>",
		Aliases: []string{"rm"},
		Short:   "Delete a stash entry",
		Long:    helpdocs.MustRender("stash/delete"),
		Args:    cobra.ExactArgs(1),
		RunE:    runStashDelete,
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Skip the confirmation prompt")

	listCmd := &cobra.Command{
		Use:     "list <scope>",
		Aliases: []string{"ls"},
		Short:   "List the stash entries in a scope",
		Long:    helpdocs.MustRender("stash/list"),
		Args:    cobra.ExactArgs(1),
		RunE:    runStashList,
	}
	listCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	stashCmd.AddCommand(getCmd, setCmd, deleteCmd, listCmd)
	return stashCmd
}

func newStashAPI(cmd *cobra.Command) (stash.API, error) {
	mdClient, err := newMassdriverClient(cmd)
	if err != nil {
		return nil, fmt.Errorf("error initializing massdriver client: %w", err)
	}
	return stash.NewAPI(mdClient), nil
}

func runStashGet(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	cmd.SilenceUsage = true

	api, err := newStashAPI(cmd)
	if err != nil {
		return err
	}
	return stash.RunGet(context.Background(), api, args[0], output, os.Stdout)
}

func runStashSet(cmd *cobra.Command, args []string) error {
	opts := stash.SetOptions{}
	opts.Format, _ = cmd.Flags().GetString("format")
	opts.TTL, _ = cmd.Flags().GetInt("ttl")
	opts.Immutable, _ = cmd.Flags().GetBool("immutable")
	opts.Sensitive, _ = cmd.Flags().GetBool("sensitive")
	cmd.SilenceUsage = true

	api, err := newStashAPI(cmd)
	if err != nil {
		return err
	}
	return stash.RunSet(context.Background(), api, args[0], args[1], opts, os.Stdin, os.Stdout)
}

func runStashDelete(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	cmd.SilenceUsage = true

	api, err := newStashAPI(cmd)
	if err != nil {
		return err
	}
	return stash.RunDelete(context.Background(), api, args[0], force, os.Stdin, os.Stdout)
}

func runStashList(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	cmd.SilenceUsage = true

	api, err := newStashAPI(cmd)
	if err != nil {
		return err
	}
	return stash.RunList(context.Background(), api, args[0], output, os.Stdout)
}
