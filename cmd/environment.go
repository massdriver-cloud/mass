package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/cli"
	"github.com/massdriver-cloud/mass/pkg/commands/environment"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/environment.get.md.tmpl
var environmentTemplates embed.FS

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

	environmentCmd.AddCommand(environmentExportCmd)
	environmentCmd.AddCommand(environmentGetCmd)
	environmentCmd.AddCommand(environmentListCmd)

	return environmentCmd
}

func runEnvironmentExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentId := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	return environment.RunExport(ctx, mdClient, environmentId)
}

func runEnvironmentGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentId := args[0]

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	environment, err := api.GetEnvironment(ctx, mdClient, environmentId)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(environment, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal environment to JSON: %w", err)
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

	projectId := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	environments, err := api.GetEnvironmentsByProject(ctx, mdClient, projectId)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID/Slug", "Name", "Description", "Monthly $", "Daily $")

	for _, env := range environments {
		tbl.AddRow(env.Slug, env.Name, env.Description, env.Cost.Monthly.Average.Amount, env.Cost.Daily.Average.Amount)
	}

	tbl.Print()

	return nil
}

func renderEnvironment(environment *api.Environment) error {
	tmplBytes, err := environmentTemplates.ReadFile("templates/environment.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("environment").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, environment); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
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
