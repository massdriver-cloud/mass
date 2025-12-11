package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

func NewCmdSchema() *cobra.Command {
	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage JSON Schemas",
		Long:  helpdocs.MustRender("schema"),
	}

	schemaDereferenceCmd := &cobra.Command{
		Use:   "dereference",
		Short: "Dereferences a JSON Schema",
		Long:  helpdocs.MustRender("schema/dereference"),
		RunE:  runSchemaDereference,
	}
	schemaDereferenceCmd.Flags().StringP("file", "f", "", "Path to JSON document")
	schemaDereferenceCmd.MarkFlagRequired("file")

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

	schemaCmd.AddCommand(schemaDereferenceCmd)
	schemaCmd.AddCommand(schemaValidateCmd)

	return schemaCmd
}

func runSchemaDereference(cmd *cobra.Command, args []string) error {
	schemaPath, _ := cmd.Flags().GetString("file")
	cmd.SilenceUsage = true

	schemaFile, openErr := os.Open(schemaPath)
	if openErr != nil {
		return fmt.Errorf("failed to open schema file %s: %w", schemaPath, openErr)
	}
	defer schemaFile.Close()
	basePath := filepath.Dir(schemaPath)

	var rawSchema map[string]any
	if err := json.NewDecoder(schemaFile).Decode(&rawSchema); err != nil {
		return fmt.Errorf("failed to decode JSON schema: %w", err)
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	derefOpts := definition.DereferenceOptions{
		Client: mdClient,
		Cwd:    basePath,
	}
	dereferencedSchema, derefErr := definition.DereferenceSchema(rawSchema, derefOpts)
	if derefErr != nil {
		return fmt.Errorf("failed to dereference schema: %w", derefErr)
	}

	dereferencedJSON, jsonErr := json.MarshalIndent(dereferencedSchema, "", "  ")
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal dereferenced schema to JSON: %w", jsonErr)
	}

	fmt.Println(string(dereferencedJSON))

	return nil
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
