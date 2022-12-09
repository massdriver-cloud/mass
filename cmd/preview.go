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
var previewDeployCiContextPath string
var previewInitCmdHelp = mustRender(`# TODO`)
var previewDeployCmdHelp = mustRender(`# TODO`)

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

var previewDeployCmd = &cobra.Command{
	Use:   "deploy projectSlug",
	Short: "Deploys a preview environment in your project",
	Long:  previewDeployCmdHelp,
	RunE:  runPreviewDeploy,
	Args:  cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(previewCmd)

	previewInitCmd.Flags().StringVarP(&previewInitParamsPath, "output", "o", "./preview.json", "Output path for preview environment params file. This file supports bash interpolation and can be manually edited or programatically modified during CI.")
	previewCmd.AddCommand(previewInitCmd)

	previewCmd.AddCommand(previewDeployCmd)
	previewDeployCmd.Flags().StringVarP(&previewInitParamsPath, "params", "p", "./preview.json", "Path to preview params file. This file supports bash interpolation.")
	previewDeployCmd.Flags().StringVarP(&previewDeployCiContextPath, "ci-context", "c", "", "Path to GitHub Actions event.json")
}

func runPreviewInit(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	// TODO: send stdin
	cfg, err := commands.InitializePreviewEnvironment(client, c.OrgID, projectSlug)

	if err != nil {
		return err
	}

	return files.Write(previewInitParamsPath, cfg)
}

func runPreviewDeploy(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)

	// TODO: parse and pass in files...
	_, err := commands.DeployPreviewEnvironment(client, c.OrgID, projectSlug)

	return err
}
