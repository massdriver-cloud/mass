package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/mitchellh/mapstructure"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

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
		Args:  cobra.ExactArgs(1),
		RunE:  runDefinitionGet,
	}

	definitionListCmd := &cobra.Command{
		Use:   "list [definition]",
		Short: "List artifact definitions",
		RunE:  runDefinitionList,
	}

	definitionPublishCmd := &cobra.Command{
		Use:          "publish",
		Short:        "Publish an artifact definition to Massdriver",
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
	definitionName := args[0]

	c := restclient.NewClient()

	adMap, err := definition.Get(c, definitionName)
	if err != nil {
		return err
	}

	// Convert map to ArtifactDefinitionWithSchema
	var ad api.ArtifactDefinitionWithSchema
	if err := mapstructure.Decode(adMap, &ad); err != nil {
		return fmt.Errorf("failed to decode definition: %w", err)
	}

	err = renderDefinition(&ad)

	return err
}

func runDefinitionPublish(cmd *cobra.Command, args []string) error {
	defPath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	c := restclient.NewClient()

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

	if pubErr := definition.Publish(c, defFile); pubErr != nil {
		return pubErr
	}

	fmt.Println("Definition published successfully!")

	return nil
}

func runDefinitionList(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	client := api.NewClient(config.URL, config.APIKey)
	definitions, err := api.ListArtifactDefinitions(client, config.OrgID)

	headerFmt := color.New(color.FgHiBlue, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgHiWhite).SprintfFunc()

	tbl := table.New("ID", "Label", "Updated At")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

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

	md := fmt.Sprintf(`# Artifact Definition: %s

**ID:** %s

## Schema
`+"```json"+`
%s
`+"```"+`
`, ad.Label, ad.Name, string(schemaJSON))

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(md)
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}
