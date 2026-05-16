package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/environments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
)

//go:embed templates/environment.get.md.tmpl
var environmentTemplates embed.FS

// NewCmdEnvironment returns a cobra command for managing environments.
func NewCmdEnvironment() *cobra.Command {
	environmentCmd := &cobra.Command{
		Use:     "environment",
		Short:   "Environment management",
		Long:    helpdocs.MustRender("environment"),
		Aliases: []string{"env"},
	}

	environmentExportCmd := &cobra.Command{
		Use:   "export [environment]",
		Short: "Export an environment from Massdriver",
		Long:  helpdocs.MustRender("environment/export"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentExport,
	}

	environmentGetCmd := &cobra.Command{
		Use:   "get [environment]",
		Short: "Get an environment from Massdriver",
		Long:  helpdocs.MustRender("environment/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentGet,
	}
	environmentGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	environmentListCmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Long:  helpdocs.MustRender("environment/list"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentList,
	}

	environmentCreateCmd := &cobra.Command{
		Use:   "create [ID]",
		Short: "Create an environment",
		Long:  helpdocs.MustRender("environment/create"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentCreate,
	}
	environmentCreateCmd.Flags().StringP("name", "n", "", "Environment name (defaults to ID if not provided)")
	environmentCreateCmd.Flags().StringP("description", "d", "", "Optional environment description")
	environmentCreateCmd.Flags().StringToStringP("attributes", "a", nil, "Custom attributes for ABAC (e.g. -a environment=staging,region=uswest)")

	environmentUpdateCmd := &cobra.Command{
		Use:   "update [environment]",
		Short: "Update an environment's name, description, or attributes",
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentUpdate,
	}
	environmentUpdateCmd.Flags().StringP("name", "n", "", "New environment name")
	environmentUpdateCmd.Flags().StringP("description", "d", "", "New environment description")
	environmentUpdateCmd.Flags().StringToStringP("attributes", "a", nil, "Replacement custom attributes (e.g. -a environment=staging,region=uswest)")

	environmentDefaultCmd := &cobra.Command{
		Use:   "default [environment] [resource-id]",
		Short: "Set an environment default connection",
		Long:  helpdocs.MustRender("environment/default"),
		Args:  cobra.ExactArgs(2),
		RunE:  runEnvironmentDefault,
	}

	environmentCmd.AddCommand(environmentExportCmd)
	environmentCmd.AddCommand(environmentGetCmd)
	environmentCmd.AddCommand(environmentListCmd)
	environmentCmd.AddCommand(environmentCreateCmd)
	environmentCmd.AddCommand(environmentUpdateCmd)
	environmentCmd.AddCommand(environmentDefaultCmd)
	environmentCmd.AddCommand(newEnvironmentPreviewCmd())
	environmentCmd.AddCommand(newEnvironmentForkCmd())
	environmentCmd.AddCommand(newEnvironmentDeployCmd())

	return environmentCmd
}

func newEnvironmentPreviewCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "preview [ID]",
		Short: "Converge a preview environment from a YAML config",
		Long:  helpdocs.MustRender("environment/preview"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvironmentPreview,
	}
	c.Flags().StringP("file", "f", "preview.yaml", "Path to the preview config YAML")
	c.Flags().StringP("name", "n", "", "Environment name (defaults to ID if not provided)")
	c.Flags().StringP("description", "d", "", "Optional environment description")
	c.Flags().StringToStringP("attributes", "a", nil, "Custom attributes for ABAC (e.g. -a environment=preview,region=uswest). Overrides `attributes:` in the config file.")
	c.Flags().Bool("follow", false, "Stream every deployment's logs to stdout until the rollout completes. Each line is prefixed with the instance id.")
	return c
}

func newEnvironmentForkCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "fork [parent-environment] [new-ID]",
		Short:   "Fork an existing environment",
		Example: `mass environment fork ecomm-production staging`,
		Long:    helpdocs.MustRender("environment/fork"),
		Args:    cobra.ExactArgs(2),
		RunE:    runEnvironmentFork,
	}
	c.Flags().StringP("name", "n", "", "Environment name (defaults to new-ID if not provided)")
	c.Flags().StringP("description", "d", "", "Optional environment description")
	c.Flags().StringToStringP("attributes", "a", nil, "Custom attributes for ABAC (e.g. -a region=uswest)")
	c.Flags().Bool("copy-environment-defaults", false, "Copy the parent's default resource connections into the fork")
	c.Flags().Bool("copy-secrets", false, "Copy every instance's secrets from the parent into the fork")
	c.Flags().Bool("copy-remote-references", false, "Copy every instance's remote references from the parent into the fork")
	return c
}

func newEnvironmentDeployCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "deploy [environment]",
		Short:   "Deploy every instance in an environment, in dependency order",
		Example: `mass environment deploy ecomm-staging --follow`,
		Long:    helpdocs.MustRender("environment/deploy"),
		Args:    cobra.ExactArgs(1),
		RunE:    runEnvironmentDeploy,
	}
	c.Flags().Bool("follow", false, "Stream every deployment's logs to stdout until the rollout completes. Each line is prefixed with the instance id.")
	return c
}

func runEnvironmentExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return environment.RunExport(ctx, mdClient, environmentID)
}

func runEnvironmentGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	env, err := mdClient.Environments.Get(ctx, environmentID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(env, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal environment to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		if err := renderEnvironment(ctx, mdClient, env); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runEnvironmentList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectID := args[0]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	envs, err := mdClient.Environments.List(ctx, environments.ListInput{ProjectID: projectID})
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Description", "Monthly $", "Daily $")

	for _, env := range envs {
		monthly := ""
		daily := ""
		if env.Cost.MonthlyAverage.Amount != nil {
			monthly = fmt.Sprintf("%v", *env.Cost.MonthlyAverage.Amount)
		}
		if env.Cost.DailyAverage.Amount != nil {
			daily = fmt.Sprintf("%v", *env.Cost.DailyAverage.Amount)
		}
		description := cli.TruncateString(env.Description, 60)
		tbl.AddRow(env.ID, env.Name, description, monthly, daily)
	}

	tbl.Print()

	return nil
}

func renderEnvironment(ctx context.Context, mdClient *massdriver.Client, env *environments.Environment) error {
	tmplBytes, err := environmentTemplates.ReadFile("templates/environment.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("environment").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Instances aren't embedded on the environment record returned by
	// Environments.Get; fetch them separately so the template can render them.
	insts, err := mdClient.Instances.List(ctx, instances.ListInput{EnvironmentID: env.ID})
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	data := struct {
		*types.Environment
		Instances []types.Instance
	}{Environment: env, Instances: insts}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, data); renderErr != nil {
		return fmt.Errorf("failed to execute template: %w", renderErr)
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(buf.String())
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}

func runEnvironmentCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fullID := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}

	// Parse project-env format: extract project and env IDs separately
	parts := strings.Split(fullID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("unable to determine project from ID %s (expected format: project-env)", fullID)
	}
	projectID := parts[0]
	envID := strings.Join(parts[1:], "-")

	if name == "" {
		name = envID
	}

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	input := environments.CreateInput{
		ID:          envID,
		Name:        name,
		Description: description,
		Attributes:  cli.AttributesToAnyMap(attrs),
	}

	env, err := mdClient.Environments.Create(ctx, projectID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Environment `%s` created successfully\n", fullID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(env.ID))
	return nil
}

//nolint:dupl // parallel CRUD shape with other entity update commands
func runEnvironmentUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	current, err := mdClient.Environments.Get(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("error getting environment: %w", err)
	}

	if !cmd.Flags().Changed("name") {
		name = current.Name
	}
	if !cmd.Flags().Changed("description") {
		description = current.Description
	}
	var attributes map[string]any
	if cmd.Flags().Changed("attributes") {
		attributes = cli.AttributesToAnyMap(attrs)
	} else {
		attributes = current.Attributes
	}

	input := environments.UpdateInput{
		Name:        name,
		Description: description,
		Attributes:  attributes,
	}

	updated, err := mdClient.Environments.Update(ctx, environmentID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Environment `%s` updated\n", updated.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(updated.ID))
	return nil
}

func runEnvironmentDefault(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]
	resourceID := args[1]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	if _, setErr := mdClient.Environments.SetDefault(ctx, environmentID, resourceID); setErr != nil {
		return setErr
	}

	env, err := mdClient.Environments.Get(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	fmt.Printf("✅ Environment `%s` default connection set successfully\n", env.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(env.ID))

	return nil
}

func runEnvironmentPreview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	id := args[0]
	configPath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	config, configErr := environment.LoadPreviewConfig(configPath)
	if configErr != nil {
		return configErr
	}

	mdClient, mdClientErr := massdriver.NewClient()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	previewOpts := environment.PreviewOptions{
		ID:          id,
		Name:        name,
		Description: description,
	}
	if cmd.Flags().Changed("attributes") {
		previewOpts.Attributes = attrs
	}

	env, runErr := environment.RunPreview(ctx, environment.NewPreviewAPI(mdClient), config, previewOpts)
	if runErr != nil {
		return runErr
	}

	fmt.Printf("✅ Preview environment `%s` converged\n", env.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(env.ID))

	if follow {
		return environment.FollowEnvironment(ctx, environment.NewFollowAPI(mdClient), env.ID, os.Stdout)
	}
	return nil
}

func runEnvironmentFork(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	parentID := args[0]
	newLocalID := args[1]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}
	copyDefaults, err := cmd.Flags().GetBool("copy-environment-defaults")
	if err != nil {
		return err
	}
	copySecrets, err := cmd.Flags().GetBool("copy-secrets")
	if err != nil {
		return err
	}
	copyRefs, err := cmd.Flags().GetBool("copy-remote-references")
	if err != nil {
		return err
	}

	if name == "" {
		name = newLocalID
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := massdriver.NewClient()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := environments.ForkInput{
		ID:                      newLocalID,
		Name:                    name,
		Description:             description,
		Attributes:              cli.AttributesToAnyMap(attrs),
		CopyEnvironmentDefaults: copyDefaults,
		CopySecrets:             copySecrets,
		CopyRemoteReferences:    copyRefs,
	}

	env, err := mdClient.Environments.Fork(ctx, parentID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Environment `%s` forked from `%s`\n", env.ID, parentID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(env.ID))
	return nil
}

func runEnvironmentDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := massdriver.NewClient()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	env, err := mdClient.Environments.Deploy(ctx, environmentID)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Deploying environment `%s` — instances roll out in dependency order asynchronously\n", env.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).EnvironmentURL(env.ID))

	if follow {
		return environment.FollowEnvironment(ctx, environment.NewFollowAPI(mdClient), env.ID, os.Stdout)
	}
	return nil
}
