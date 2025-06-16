package cmd

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

//go:embed templates/project.get.md.tmpl
var projectTemplates embed.FS

func NewCmdProject() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:     "project",
		Short:   "Project management",
		Long:    helpdocs.MustRender("project"),
		Aliases: []string{"prj"},
	}

	projectGetCmd := &cobra.Command{
		Use:   "get [project]",
		Short: "Get a project from Massdriver",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectGet,
	}
	projectGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	projectListCmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE:  runProjectList,
	}

	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectListCmd)

	return projectCmd
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	config, err := config.Get()
	if err != nil {
		return err
	}

	client := api.NewClient(config.URL, config.APIKey)
	projects, err := api.ListProjects(client, config.OrgID)
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

func runProjectList(cmd *cobra.Command, args []string) error {
	config, err := config.Get()
	if err != nil {
		return err
	}

	client := api.NewClient(config.URL, config.APIKey)
	projects, err := api.ListProjects(client, config.OrgID)
	if err != nil {
		return err
	}

	headerFmt := color.New(color.FgHiBlue, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgHiWhite).SprintfFunc()

	tbl := table.New("Name", "Description")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, project := range projects {
		tbl.AddRow(project.Name, project.Description)
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
