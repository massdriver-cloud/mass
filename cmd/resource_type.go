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
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/spf13/cobra"
)

//go:embed templates/type.get.md.tmpl
var typeTemplates embed.FS

// NewCmdType returns a cobra command for managing resource types.
func NewCmdType() *cobra.Command {
	typeCmd := &cobra.Command{
		Use:     "resource-type",
		Short:   "Resource type management",
		Long:    helpdocs.MustRender("type"),
		Aliases: []string{"rt", "type", "res-type", "definition", "artifact-definition", "artdef", "def"},
	}

	typeGetCmd := &cobra.Command{
		Use:   "get [resource-type]",
		Short: "Get a resource type from Massdriver",
		Long:  helpdocs.MustRender("type/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypeGet,
	}
	typeGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	typeListCmd := &cobra.Command{
		Use:   "list",
		Short: "List resource types",
		Long:  helpdocs.MustRender("type/list"),
		RunE:  runTypeList,
	}

	typePublishCmd := &cobra.Command{
		Use:   "publish [resource-type file]",
		Short: "Publish a resource type to Massdriver",
		Long:  helpdocs.MustRender("type/publish"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypePublish,
	}

	typeDeleteCmd := &cobra.Command{
		Use:   "delete [resource-type]",
		Short: "Delete a resource type from Massdriver",
		Long:  helpdocs.MustRender("type/delete"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypeDelete,
	}
	typeDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	typeCmd.AddCommand(typeGetCmd)
	typeCmd.AddCommand(typePublishCmd)
	typeCmd.AddCommand(typeListCmd)
	typeCmd.AddCommand(typeDeleteCmd)

	return typeCmd
}

//nolint:dupl // runTypeGet and runResourceGet share the same output-format pattern; refactoring would add complexity for marginal gain
func runTypeGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	typeName := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	rt, err := resourcetype.Get(ctx, mdClient, typeName)
	if err != nil {
		return fmt.Errorf("error getting resource type: %w", err)
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(rt, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resource type to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderType(rt)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runTypePublish(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	defFile := args[0]
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	artDef, publishErr := resourcetype.Publish(ctx, mdClient, defFile)
	if publishErr != nil {
		return fmt.Errorf("error publishing resource type: %w", publishErr)
	}

	fmt.Printf("Resource type %s published successfully!\n", prettylogs.Underline(artDef.Name))

	return nil
}

func runTypeList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	resourceTypes, err := resourcetype.List(ctx, mdClient)

	tbl := cli.NewTable("ID", "Name", "Updated At")

	for _, rt := range resourceTypes {
		tbl.AddRow(rt.ID, rt.Name, rt.UpdatedAt)
	}

	tbl.Print()

	return err
}

func renderType(restype *resourcetype.ResourceType) error {
	schemaJSON, err := json.MarshalIndent(restype.Schema, "", "  ")
	if err != nil {
		return err
	}

	tmplBytes, err := typeTemplates.ReadFile("templates/type.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("type").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ID         string
		Name       string
		SchemaJSON string
	}{
		ID:         restype.ID,
		Name:       restype.Name,
		SchemaJSON: string(schemaJSON),
	}

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

func runTypeDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	typeName := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	// Get resource type details for confirmation
	rt, err := resourcetype.Get(ctx, mdClient, typeName)
	if err != nil {
		return fmt.Errorf("error getting resource type: %w", err)
	}

	// Prompt for confirmation - requires typing the resource type name unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete resource type `%s`.\n", rt.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", rt.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != rt.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deleted, deleteErr := resourcetype.Delete(ctx, mdClient, typeName)
	if deleteErr != nil {
		return fmt.Errorf("error deleting resource type: %w", deleteErr)
	}

	fmt.Printf("Resource type %s deleted successfully!\n", prettylogs.Underline(deleted.Name))
	return nil
}
