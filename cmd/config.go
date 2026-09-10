package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	cmdconfig "github.com/massdriver-cloud/mass/internal/commands/config"
	"github.com/massdriver-cloud/mass/internal/configfile"
	"github.com/spf13/cobra"
)

// NewCmdConfig returns a cobra command for managing the CLI configuration file.
func NewCmdConfig() *cobra.Command {
	configCmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage CLI configuration profiles",
		Long:    helpdocs.MustRender("config"),
		Aliases: []string{"profile"},
	}

	configListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured profiles",
		Example: `mass config list`,
		Long:    helpdocs.MustRender("config/list"),
		Args:    cobra.NoArgs,
		RunE:    runConfigList,
	}
	configListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	configListCmd.Flags().Bool("show-secrets", false, "Print API keys in full instead of masking them")

	configGetCmd := &cobra.Command{
		Use:     "get [profile]",
		Short:   "Show a profile's settings (defaults to the active profile)",
		Example: `mass config get staging`,
		Long:    helpdocs.MustRender("config/get"),
		Args:    cobra.MaximumNArgs(1),
		RunE:    runConfigGet,
	}
	configGetCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	configGetCmd.Flags().Bool("show-secrets", false, "Print the API key in full instead of masking it")

	configAddCmd := &cobra.Command{
		Use:     "add [profile]",
		Aliases: []string{"create"},
		Short:   "Add a profile",
		Example: `mass config add staging --org acme --api-key mds_xxx`,
		Long:    helpdocs.MustRender("config/add"),
		Args:    cobra.ExactArgs(1),
		RunE:    runConfigAdd,
	}
	configAddCmd.Flags().String("org", "", "Organization abbreviation")
	configAddCmd.Flags().String("api-key", "", "API key or personal access token")
	configAddCmd.Flags().String("url", "", "Massdriver API URL (defaults to https://api.massdriver.cloud)")
	configAddCmd.Flags().String("templates-path", "", "Directory containing bundle templates")
	configAddCmd.Flags().BoolP("force", "f", false, "Overwrite the profile if it already exists")
	configAddCmd.Flags().Bool("no-verify", false, "Skip checking the credentials against the Massdriver API")
	configAddCmd.Flags().Bool("use", false, "Make this the active profile after adding it")

	configSetCmd := &cobra.Command{
		Use:     "set [profile]",
		Aliases: []string{"update"},
		Short:   "Update settings on an existing profile",
		Example: `mass config set staging --url https://api.massdriver.cloud`,
		Long:    helpdocs.MustRender("config/set"),
		Args:    cobra.ExactArgs(1),
		RunE:    runConfigSet,
	}
	configSetCmd.Flags().String("org", "", "Organization abbreviation")
	configSetCmd.Flags().String("api-key", "", "API key or personal access token")
	configSetCmd.Flags().String("url", "", "Massdriver API URL (defaults to https://api.massdriver.cloud)")
	configSetCmd.Flags().String("templates-path", "", "Directory containing bundle templates")
	configSetCmd.Flags().Bool("no-verify", false, "Skip checking the credentials against the Massdriver API")

	configRemoveCmd := &cobra.Command{
		Use:     "remove [profile]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a profile",
		Example: `mass config remove staging`,
		Long:    helpdocs.MustRender("config/remove"),
		Args:    cobra.ExactArgs(1),
		RunE:    runConfigRemove,
	}
	configRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	configUseCmd := &cobra.Command{
		Use:     "use [profile]",
		Short:   "Set the active profile",
		Example: `mass config use staging`,
		Long:    helpdocs.MustRender("config/use"),
		Args:    cobra.ExactArgs(1),
		RunE:    runConfigUse,
	}

	configPathCmd := &cobra.Command{
		Use:     "path",
		Short:   "Print the path to the configuration file",
		Example: `mass config path`,
		Long:    helpdocs.MustRender("config/path"),
		Args:    cobra.NoArgs,
		RunE:    runConfigPath,
	}

	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configPathCmd)

	return configCmd
}

func runConfigList(cmd *cobra.Command, args []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	showSecrets, err := cmd.Flags().GetBool("show-secrets")
	if err != nil {
		return err
	}
	profileOverride, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	opts := cmdconfig.ListOptions{Output: output, ShowSecrets: showSecrets, ProfileOverride: profileOverride}
	return cmdconfig.RunList(file, opts, os.Stdout)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	showSecrets, err := cmd.Flags().GetBool("show-secrets")
	if err != nil {
		return err
	}
	profileOverride, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}

	var name string
	if len(args) == 1 {
		name = args[0]
	}

	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	opts := cmdconfig.GetOptions{Output: output, ShowSecrets: showSecrets, ProfileOverride: profileOverride}
	return cmdconfig.RunGet(file, name, opts, os.Stdout)
}

func runConfigAdd(cmd *cobra.Command, args []string) error {
	edits, err := profileEdits(cmd)
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	use, err := cmd.Flags().GetBool("use")
	if err != nil {
		return err
	}
	noVerify, err := cmd.Flags().GetBool("no-verify")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	opts := cmdconfig.AddOptions{
		Edits:    edits,
		Force:    force,
		Use:      use,
		Prompter: cmdconfig.NewPrompter(),
	}
	if !noVerify {
		opts.Verifier = cmdconfig.NewVerifier()
	}
	return cmdconfig.RunAdd(context.Background(), file, args[0], opts, os.Stdout)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	edits, err := profileEdits(cmd)
	if err != nil {
		return err
	}
	noVerify, err := cmd.Flags().GetBool("no-verify")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	var verifier cmdconfig.Verifier
	if !noVerify {
		verifier = cmdconfig.NewVerifier()
	}
	return cmdconfig.RunSet(context.Background(), file, args[0], edits, verifier, os.Stdout)
}

func runConfigRemove(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	return cmdconfig.RunRemove(file, args[0], force, os.Stdin, os.Stdout)
}

func runConfigUse(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	file, err := configfile.Load()
	if err != nil {
		return err
	}

	return cmdconfig.RunUse(file, args[0], os.Stdout)
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	path, err := configfile.Path()
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// profileEdits reads the value flags `add` and `set` share, recording only the
// ones actually passed so an update leaves the rest of the profile alone.
func profileEdits(cmd *cobra.Command) (cmdconfig.ProfileEdits, error) {
	var edits cmdconfig.ProfileEdits
	for _, field := range []struct {
		flag string
		dest **string
	}{
		{"org", &edits.OrganizationID},
		{"api-key", &edits.APIKey},
		{"url", &edits.URL},
		{"templates-path", &edits.TemplatesPath},
	} {
		if !cmd.Flags().Changed(field.flag) {
			continue
		}
		value, err := cmd.Flags().GetString(field.flag)
		if err != nil {
			return edits, err
		}
		*field.dest = &value
	}
	return edits, nil
}
