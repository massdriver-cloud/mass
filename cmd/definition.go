package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/cli"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/definition.get.md.tmpl
var definitionTemplates embed.FS

func NewCmdDefinition() *cobra.Command {
	definitionCmd := &cobra.Command{
		Use:     "definition",
		Short:   "Artifact definition management",
		Long:    helpdocs.MustRender("definition"),
		Aliases: []string{"artifact-definition", "artdef", "def"},
	}

	definitionGetCmd := &cobra.Command{
		Use:   "get [definition]",
		Short: "Get an artifact definition from Massdriver",
		Long:  helpdocs.MustRender("definition/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runDefinitionGet,
	}
	definitionGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	definitionListCmd := &cobra.Command{
		Use:   "list [definition]",
		Short: "List artifact definitions",
		Long:  helpdocs.MustRender("definition/list"),
		RunE:  runDefinitionList,
	}

	definitionPublishCmd := &cobra.Command{
		Use:          "publish",
		Short:        "Publish an artifact definition to Massdriver",
		Long:         helpdocs.MustRender("definition/publish"),
		RunE:         runDefinitionPublish,
		SilenceUsage: true,
	}
	definitionPublishCmd.Flags().StringP("file", "f", "", "File containing artifact definition schema (use - for stdin)")
	_ = definitionPublishCmd.MarkFlagRequired("file")

	definitionCmd.AddCommand(definitionGetCmd)
	definitionCmd.AddCommand(definitionPublishCmd)
	definitionCmd.AddCommand(definitionListCmd)

	return definitionCmd
}

func runDefinitionGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	definitionName := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

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

	defPath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	var defFile *os.File
	if defPath == "-" {
		defFile = os.Stdin
	} else {
		defFile, err = os.Open(defPath)
		if err != nil {
			fmt.Println(err)
		}
		defer defFile.Close()
	}

	defBytes, readErr := io.ReadAll(defFile)
	if readErr != nil {
		return fmt.Errorf("error reading artifact definition: %w", readErr)
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// TODO: All the logic from here to Publish should be moved into the commands package
	def := map[string]any{}
	jsonErr := json.Unmarshal(defBytes, &def)
	if jsonErr != nil {
		return jsonErr
	}

	mdBlock, mdExists := def["$md"].(map[string]any)
	if !mdExists {
		return fmt.Errorf("artifact definition must have a '$md' block with a 'name' field")
	}
	_, nameExists := mdBlock["name"].(string)
	if !nameExists {
		return fmt.Errorf("artifact definition must have a 'name' field")
	}

	_, publishErr := api.PublishArtifactDefinition(ctx, mdClient, def)
	if publishErr != nil {
		return fmt.Errorf("error publishing artifact definition: %w", publishErr)
	}

	fmt.Println("Definition published successfully!")

	return nil
}

func runDefinitionList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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
