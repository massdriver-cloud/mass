package cmd

import (
	"bufio"
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
	"github.com/massdriver-cloud/mass/internal/commands/project"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/projects"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
)

//go:embed templates/project.get.md.tmpl
var projectTemplates embed.FS

// NewCmdProject returns a cobra command for managing projects.
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

	projectCreateCmd := &cobra.Command{
		Use:   "create [slug]",
		Short: "Create a project",
		Long:  helpdocs.MustRender("project/create"),
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectCreate,
	}
	projectCreateCmd.Flags().StringP("name", "n", "", "Project name (defaults to slug if not provided)")
	projectCreateCmd.Flags().StringToStringP("attributes", "a", nil, "Custom attributes for ABAC (repeat or comma-separate, e.g. -a team=ops,system=api)")
	projectCreateCmd.Flags().StringP("description", "d", "", "Optional project description")

	projectUpdateCmd := &cobra.Command{
		Use:   "update [project]",
		Short: "Update a project's name, description, or attributes",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectUpdate,
	}
	projectUpdateCmd.Flags().StringP("name", "n", "", "New project name")
	projectUpdateCmd.Flags().StringP("description", "d", "", "New project description")
	projectUpdateCmd.Flags().StringToStringP("attributes", "a", nil, "Replacement custom attributes (e.g. -a team=ops,system=api)")

	projectDeleteCmd := &cobra.Command{
		Use:   "delete [project]",
		Short: "Delete a project",
		Long:  helpdocs.MustRender("project/delete"),
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectDelete,
	}
	projectDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	projectCmd.AddCommand(projectExportCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	return projectCmd
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectID := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	proj, err := mdClient.Projects.Get(ctx, projectID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(proj, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal project to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderProject(proj)
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

	projectID := args[0]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return project.RunExport(ctx, mdClient, projectID)
}

func runProjectList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	projectList, err := mdClient.Projects.List(ctx)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID/Slug", "Name", "Description", "Monthly $", "Daily $")

	for _, p := range projectList {
		monthly := ""
		daily := ""
		if p.Cost != nil {
			if p.Cost.MonthlyAverage.Amount != nil {
				monthly = fmt.Sprintf("%v", *p.Cost.MonthlyAverage.Amount)
			}
			if p.Cost.DailyAverage.Amount != nil {
				daily = fmt.Sprintf("%v", *p.Cost.DailyAverage.Amount)
			}
		}
		description := cli.TruncateString(p.Description, 60)
		tbl.AddRow(p.ID, p.Name, description, monthly, daily)
	}

	tbl.Print()

	return nil
}

func renderProject(p *projects.Project) error {
	tmplBytes, err := projectTemplates.ReadFile("templates/project.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("project").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// The template chains through .Cost.MonthlyAverage.Amount unconditionally;
	// supply a zero CostSummary so a nil cost record renders as empty values
	// rather than aborting template execution.
	if p.Cost == nil {
		p.Cost = &types.CostSummary{}
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, p); renderErr != nil {
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

func runProjectCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	id := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	if name == "" {
		name = id
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

	input := projects.CreateInput{
		ID:          id,
		Name:        name,
		Description: description,
		Attributes:  cli.AttributesToAnyMap(attrs),
	}

	proj, err := mdClient.Projects.Create(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Project `%s` created successfully\n", proj.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).ProjectURL(proj.ID))
	return nil
}

//nolint:dupl // parallel CRUD shape with other entity update commands
func runProjectUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectID := args[0]
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

	// Fetch current state so unset flags retain their existing values rather
	// than blanking the field at the server.
	current, err := mdClient.Projects.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("error getting project: %w", err)
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

	input := projects.UpdateInput{
		Name:        name,
		Description: description,
		Attributes:  attributes,
	}

	updated, err := mdClient.Projects.Update(ctx, projectID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Project `%s` updated\n", updated.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).ProjectURL(updated.ID))
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectID := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	// Get project details for confirmation
	proj, err := mdClient.Projects.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("error getting project: %w", err)
	}

	// Prompt for confirmation - requires typing the project ID unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete project `%s` and all its resources.\n", proj.ID)
		fmt.Printf("Type `%s` to confirm deletion: ", proj.ID)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != proj.ID {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deletedProject, err := mdClient.Projects.Delete(ctx, projectID)
	if err != nil {
		return err
	}

	fmt.Printf("Project %s deleted successfully (ID: %s)\n", deletedProject.Name, deletedProject.ID)
	return nil
}
