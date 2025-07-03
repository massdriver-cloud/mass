package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/spf13/cobra"
)

func NewCmdSchema() *cobra.Command {
	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage JSON Schemas",
		Long:  helpdocs.MustRender("schema"),
	}

	schemaValidateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates a JSON document against a JSON Schema",
		Long:  helpdocs.MustRender("schema/validate"),
		RunE:  runSchemaValidate,
	}
	schemaValidateCmd.Flags().StringP("document", "d", "document.json", "Path to JSON document")
	schemaValidateCmd.Flags().StringP("schema", "s", "./schema.json", "Path to JSON Schema")
	schemaValidateCmd.MarkFlagRequired("document")
	schemaValidateCmd.MarkFlagRequired("schema")

	schemaCmd.AddCommand(schemaValidateCmd)

	return schemaCmd
}

func runSchemaValidate(cmd *cobra.Command, args []string) error {
	schemaPath, _ := cmd.Flags().GetString("schema")
	documentPath, _ := cmd.Flags().GetString("document")
	cmd.SilenceUsage = true

	schema, schemaErr := jsonschema.LoadSchemaFromFile(schemaPath)
	if schemaErr != nil {
		return fmt.Errorf("failed to load schema from %s: %w", schemaPath, schemaErr)
	}

	validateErr := jsonschema.ValidateFile(schema, documentPath)
	if validateErr != nil {
		fmt.Println(prettylogs.Red(" ✗"), "The document is not valid against the schema!")
		return validateErr
	}

	greenCheckmark := prettylogs.Green(" ✓")
	fmt.Println(greenCheckmark, "The document is valid against the schema!")
	return nil
}
