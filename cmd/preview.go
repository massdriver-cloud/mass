package cmd

import (
	"fmt"

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

var previewInitParamsPath = "./preview.json"
var previewDeployCiContextPath = "/home/runner/work/_temp/_github_workflow/event.json"
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
	previewDeployCmd.Flags().StringVarP(&previewInitParamsPath, "params", "p", previewInitParamsPath, "Path to preview environment configuration file. This file supports bash interpolation.")
	previewDeployCmd.Flags().StringVarP(&previewDeployCiContextPath, "ci-context", "c", previewDeployCiContextPath, "Path to GitHub Actions event.json")
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
	previewCfg := commands.PreviewConfig{}
	ciContext := map[string]interface{}{}

	if err := files.Read(previewInitParamsPath, &previewCfg); err != nil {
		return err
	}

	if err := files.Read(previewDeployCiContextPath, &ciContext); err != nil {
		return err
	}

	env, err := commands.DeployPreviewEnvironment(client, c.OrgID, projectSlug, &previewCfg, &ciContext)

	if err != nil {
		return err
	}

	// TODO: bubbletea v zerolog
	fmt.Printf("Deploying @ %s", env.URL)

	return nil
}
