package cmd

import (
	"errors"
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"
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

	schemaCmd.AddCommand(schemaValidateCmd)

	return schemaCmd
}

func runSchemaValidate(cmd *cobra.Command, args []string) error {
	schema, _ := cmd.Flags().GetString("schema")
	document, _ := cmd.Flags().GetString("document")

	result, err := jsonschema.Validate(schema, document)
	if err != nil {
		return err
	}

	if result.Valid() {
		fmt.Println("The document is valid!")
	} else {
		errMsg := fmt.Sprintf("The document failed validation:\n\tDocument: %s\n\tSchema: %s\nErrors:\n", document, schema)
		for _, violation := range result.Errors() {
			errMsg += fmt.Sprintf("\t- %v\n", violation)
		}

		return errors.New(errMsg)
	}
	return nil
}
