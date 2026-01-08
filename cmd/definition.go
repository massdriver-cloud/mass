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
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/cli"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/spf13/cobra"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

//go:embed templates/definition.get.md.tmpl
var definitionTemplates embed.FS

func NewCmdDefinition() *cobra.Command {
	definitionCmd := &cobra.Command{
		Use:     "definition",
		Short:   "Manage artifact definitions",
		Long:    helpdocs.MustRender("definition"),
		Aliases: []string{"artifact-definition", "artdef", "def"},
	}

	definitionGetCmd := &cobra.Command{
		Use:   "get [definition]",
		Short: "Get artifact definition details",
		Long:  helpdocs.MustRender("definition/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runDefinitionGet,
		Example: `  # Get definition details
  mass definition get massdriver/aws-iam-role

  # Get definition as JSON
  mass definition get massdriver/aws-vpc --output json`,
	}
	definitionGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	definitionListCmd := &cobra.Command{
		Use:     "list [definition]",
		Short:   "List all artifact definitions",
		Long:    helpdocs.MustRender("definition/list"),
		RunE:    runDefinitionList,
		Example: `  # List all artifact definitions
  mass definition list`,
	}

	definitionPublishCmd := &cobra.Command{
		Use:   "publish [definition file]",
		Short: "Publish artifact definition",
		Long:  helpdocs.MustRender("definition/publish"),
		Args:  cobra.ExactArgs(1),
		RunE:  runDefinitionPublish,
		Example: `  # Publish a new artifact definition
  mass definition publish ./my-definition.json`,
	}

	definitionDeleteCmd := &cobra.Command{
		Use:   "delete [definition]",
		Short: "Delete artifact definition",
		Long:  helpdocs.MustRender("definition/delete"),
		Args:  cobra.ExactArgs(1),
		RunE:  runDefinitionDelete,
		Example: `  # Delete with confirmation prompt
  mass definition delete my-custom-definition

  # Delete without confirmation
  mass definition delete my-custom-definition --force`,
	}
	definitionDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	definitionCmd.AddCommand(definitionGetCmd)
	definitionCmd.AddCommand(definitionPublishCmd)
	definitionCmd.AddCommand(definitionListCmd)
	definitionCmd.AddCommand(definitionDeleteCmd)

	return definitionCmd
}

func runDefinitionGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	definitionName := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	ad, getErr := definition.Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return fmt.Errorf("error getting artifact definition: %w", getErr)
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(ad, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal definition to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderDefinition(ad)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runDefinitionPublish(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	defFile := args[0]
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	artDef, publishErr := definition.Publish(ctx, mdClient, defFile)
	if publishErr != nil {
		return fmt.Errorf("error publishing artifact definition: %w", publishErr)
	}

	fmt.Printf("Artifact definition %s published successfully!\n", prettylogs.Underline(artDef.Name))

	return nil
}

func runDefinitionList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	definitions, err := api.ListArtifactDefinitions(ctx, mdClient)

	tbl := cli.NewTable("Name", "Label", "Updated At")

	for _, definition := range definitions {
		tbl.AddRow(definition.Name, definition.Label, definition.UpdatedAt)
	}

	tbl.Print()

	return err
}

func renderDefinition(ad *api.ArtifactDefinitionWithSchema) error {
	schemaJSON, err := json.MarshalIndent(ad.Schema, "", "  ")
	if err != nil {
		return err
	}

	tmplBytes, err := definitionTemplates.ReadFile("templates/definition.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("definition").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		Label      string
		Name       string
		SchemaJSON string
	}{
		Label:      ad.Label,
		Name:       ad.Name,
		SchemaJSON: string(schemaJSON),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
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

func runDefinitionDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	definitionName := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get definition details for confirmation
	ad, getErr := definition.Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return fmt.Errorf("error getting artifact definition: %w", getErr)
	}

	// Prompt for confirmation - requires typing the definition name unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete artifact definition `%s`.\n", ad.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", ad.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != ad.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deletedDef, deleteErr := api.DeleteArtifactDefinition(ctx, mdClient, definitionName)
	if deleteErr != nil {
		return fmt.Errorf("error deleting artifact definition: %w", deleteErr)
	}

	fmt.Printf("Artifact definition %s deleted successfully!\n", prettylogs.Underline(deletedDef.Name))
	return nil
}
