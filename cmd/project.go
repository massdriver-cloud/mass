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
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/project"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	project, err := api.GetProject(ctx, mdClient, projectID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(project, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal project to JSON: %w", marshalErr)
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

	projectID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	return project.RunExport(ctx, mdClient, projectID)
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
		monthly := ""
		daily := ""
		if project.Cost.MonthlyAverage.Amount != nil {
			monthly = fmt.Sprintf("%v", *project.Cost.MonthlyAverage.Amount)
		}
		if project.Cost.DailyAverage.Amount != nil {
			daily = fmt.Sprintf("%v", *project.Cost.DailyAverage.Amount)
		}
		description := cli.TruncateString(project.Description, 60)
		tbl.AddRow(project.ID, project.Name, description, monthly, daily)
	}

	tbl.Print()

	return nil
}

func renderProject(project *api.Project) error {
	tmplBytes, err := projectTemplates.ReadFile("templates/project.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("project").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, project); renderErr != nil {
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := api.CreateProjectInput{
		Id:          id,
		Name:        name,
		Description: description,
		Attributes:  cli.AttributesToAnyMap(attrs),
	}

	project, err := api.CreateProject(ctx, mdClient, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Project `%s` created successfully\n", project.ID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.ProjectURL(project.ID))
	}
	return nil
}

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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Fetch current state so unset flags retain their existing values rather
	// than blanking the field at the server.
	current, getErr := api.GetProject(ctx, mdClient, projectID)
	if getErr != nil {
		return fmt.Errorf("error getting project: %w", getErr)
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

	input := api.UpdateProjectInput{
		Name:        name,
		Description: description,
		Attributes:  attributes,
	}

	updated, err := api.UpdateProject(ctx, mdClient, projectID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Project `%s` updated\n", updated.ID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.ProjectURL(updated.ID))
	}
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

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get project details for confirmation
	project, getErr := api.GetProject(ctx, mdClient, projectID)
	if getErr != nil {
		return fmt.Errorf("error getting project: %w", getErr)
	}

	// Prompt for confirmation - requires typing the project ID unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete project `%s` and all its resources.\n", project.ID)
		fmt.Printf("Type `%s` to confirm deletion: ", project.ID)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != project.ID {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deletedProject, err := api.DeleteProject(ctx, mdClient, projectID)
	if err != nil {
		return err
	}

	fmt.Printf("Project %s deleted successfully (ID: %s)\n", deletedProject.Name, deletedProject.ID)
	return nil
}
