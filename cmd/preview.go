package cmd

import (
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/spf13/cobra"
)

var previewCmdHelp = mustRender(`
# Preview Environments

Massdriver preview environments can deploy infrastructure and applications as a cohesive unit.
`)

var previewInitParamsPath string
var previewInitCmdHelp = mustRender(`# WIP`)

var previewCmd = &cobra.Command{
	Use:     "preview",
	Aliases: []string{"pv"},
	Short:   "Preview Environments",
	Long:    previewCmdHelp,
}

var previewInitCmd = &cobra.Command{
	Use:   `init projectSlug`,
	Short: "Generate a preview enviroment configuration file",
	Long:  previewInitCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runPreviewInit,
}

func init() {
	rootCmd.AddCommand(previewCmd)

	previewInitCmd.Flags().StringVarP(&previewInitParamsPath, "output", "o", "./preview.json", "Output path for preview environment params file. This file supports bash interpolation and can be manually edited or programatically modified during CI.")
	previewCmd.AddCommand(previewInitCmd)
}

func runPreviewInit(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	// TODO: write config to file
	// TODO: send stdin
	cfg, err := commands.InitializePreview(client, c.OrgID, projectSlug)

	if err != nil {
		return err
	}

	return files.Write(previewInitParamsPath, cfg)
}
