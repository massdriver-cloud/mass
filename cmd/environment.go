package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/environment"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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

	return environmentCmd
}

func runEnvironmentExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	environment, err := api.GetEnvironment(ctx, mdClient, environmentID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(environment, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal environment to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderEnvironment(environment)
		if err != nil {
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	filter := api.EnvironmentsFilter{
		ProjectId: &api.IdFilter{Eq: projectID},
	}

	environments, err := api.ListEnvironments(ctx, mdClient, &filter)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Description", "Monthly $", "Daily $")

	for _, env := range environments {
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

func renderEnvironment(environment *api.Environment) error {
	tmplBytes, err := environmentTemplates.ReadFile("templates/environment.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("environment").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, environment); renderErr != nil {
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := api.CreateEnvironmentInput{
		Id:          envID,
		Name:        name,
		Description: description,
		Attributes:  cli.AttributesToAnyMap(attrs),
	}

	env, err := api.CreateEnvironment(ctx, mdClient, projectID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Environment `%s` created successfully\n", fullID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.EnvironmentURL(env.ID))
	}
	return nil
}

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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	current, getErr := api.GetEnvironment(ctx, mdClient, environmentID)
	if getErr != nil {
		return fmt.Errorf("error getting environment: %w", getErr)
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
		attributes = cli.StringMapToAnyMap(current.Attributes)
	}

	input := api.UpdateEnvironmentInput{
		Name:        name,
		Description: description,
		Attributes:  attributes,
	}

	updated, err := api.UpdateEnvironment(ctx, mdClient, environmentID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Environment `%s` updated\n", updated.ID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.EnvironmentURL(updated.ID))
	}
	return nil
}

func runEnvironmentDefault(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]
	resourceID := args[1]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err := api.SetEnvironmentDefault(ctx, mdClient, environmentID, resourceID)
	if err != nil {
		return err
	}

	environment, err := api.GetEnvironment(ctx, mdClient, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	fmt.Printf("✅ Environment `%s` default connection set successfully\n", environment.ID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.EnvironmentURL(environment.ID))
	}

	return nil
}
