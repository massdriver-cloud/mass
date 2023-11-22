package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/mass/pkg/restclient"
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

	definitionPublishCmd := &cobra.Command{
		Use:          "publish",
		Short:        "Publish an artifact definition to Massdriver",
		RunE:         runDefinitionPublish,
		SilenceUsage: true,
	}
	definitionPublishCmd.Flags().StringP("file", "f", "", "File containing artifact definition schema (use - for stdin)")
	definitionPublishCmd.MarkFlagRequired("file")

	definitionCmd.AddCommand(definitionGetCmd)
	definitionCmd.AddCommand(definitionPublishCmd)

	return definitionCmd
}

func runDefinitionGet(cmd *cobra.Command, args []string) error {
	definitionName := args[0]

	c := restclient.NewClient()

	def, err := definition.Get(c, definitionName)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(def)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))

	return nil
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
