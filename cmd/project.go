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
	"github.com/massdriver-cloud/mass/pkg/commands/project"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/project.get.md.tmpl
var projectTemplates embed.FS

func NewCmdProject() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:     "project",
		Short:   "Project management",
		Long:    helpdocs.MustRender("project"),
		Aliases: []string{"prj", "proj"},
	}

	projectExportCmd := &cobra.Command{
		Use:   "export [project]",
		Short: "Export a project from Massdriver",
		Long:  helpdocs.MustRender("project/export"),
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectExport,
	}

	projectGetCmd := &cobra.Command{
		Use:   "get [project]",
		Short: "Get a project from Massdriver",
		Long:  helpdocs.MustRender("project/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectGet,
	}
	projectGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	projectListCmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  helpdocs.MustRender("project/list"),
		RunE:  runProjectList,
	}

	projectCmd.AddCommand(projectExportCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectListCmd)

	return projectCmd
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectSlug := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	projects, err := api.ListProjects(ctx, mdClient)
	if err != nil {
		return err
	}

	var project *api.Project
	for _, p := range projects {
		if p.Slug == projectSlug {
			project = &p
			break
		}
	}

	if project == nil {
		return fmt.Errorf("project not found: %s", projectSlug)
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal project to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderProject(project)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runProjectExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectId := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	return project.RunExport(ctx, mdClient, projectId)
}

func runProjectList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	projects, err := api.ListProjects(ctx, mdClient)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID/Slug", "Name", "Description", "Monthly $", "Daily $")

	for _, project := range projects {
		tbl.AddRow(project.Slug, project.Name, project.Description, project.Cost.Monthly.Average.Amount, project.Cost.Daily.Average.Amount)
	}

	tbl.Print()

	return nil
}

func renderProject(project *api.Project) error {
	tmplBytes, err := projectTemplates.ReadFile("templates/project.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("project").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, project); err != nil {
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
