package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/preview_environment/decommission"
	"github.com/massdriver-cloud/mass/internal/commands/preview_environment/deploy"
	peinit "github.com/massdriver-cloud/mass/internal/commands/preview_environment/initialize"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/spf13/cobra"
)

var previewCmdHelp = mustRenderHelpDoc("preview")
var previewInitCmdHelp = mustRenderHelpDoc("preview/init")
var previewDeployCmdHelp = mustRenderHelpDoc("preview/deploy")
var previewDecommissionCmdHelp = mustRenderHelpDoc("preview/decommission")

var previewInitParamsPath = "./preview.json"
var previewDeployCiContextPath = "/home/runner/work/_temp/_github_workflow/event.json"

var previewCmd = &cobra.Command{
	Use:     "preview",
	Aliases: []string{"pv"},
	Short:   "Create & deploy preview environments",
	Long:    previewCmdHelp,
}

var previewInitCmd = &cobra.Command{
	Use:   `init $projectSlug`,
	Short: "Generate a preview enviroment configuration file",
	Long:  previewInitCmdHelp,
	Args:  cobra.ExactArgs(1),
	RunE:  runPreviewInit,
}

var previewDeployCmd = &cobra.Command{
	Use:   "deploy $projectSlug",
	Short: "Deploys a preview environment in your project",
	Long:  previewDeployCmdHelp,
	RunE:  runPreviewDeploy,
	Args:  cobra.ExactArgs(1),
}

var previewDecommissionCmd = &cobra.Command{
	Use:   "decommission $projectTargetSlug",
	Short: "Decommissions a preview environment in your project",
	Long:  previewDecommissionCmdHelp,
	RunE:  runPreviewDecommission,
	Args:  cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(previewCmd)

	previewInitCmd.Flags().StringVarP(&previewInitParamsPath, "output", "o", "./preview.json", "Output path for preview environment params file. This file supports bash interpolation and can be manually edited or programatically modified during CI.")
	previewCmd.AddCommand(previewInitCmd)
	previewCmd.AddCommand(previewDeployCmd)
	previewCmd.AddCommand(previewDecommissionCmd)
	previewDeployCmd.Flags().StringVarP(&previewInitParamsPath, "params", "p", previewInitParamsPath, "Path to preview environment configuration file. This file supports bash interpolation.")
	previewDeployCmd.Flags().StringVarP(&previewDeployCiContextPath, "ci-context", "c", previewDeployCiContextPath, "Path to GitHub Actions event.json")
}

func runPreviewInit(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	initModel, _ := peinit.New(client, config.OrgID, projectSlug)
	p := tea.NewProgram(initModel)
	result, err := p.Run()

	if err != nil {
		return err
	}

	updatedModel, _ := (result).(peinit.Model)
	cfg := updatedModel.PreviewConfig()

	return files.Write(previewInitParamsPath, cfg)
}

func runPreviewDeploy(cmd *cobra.Command, args []string) error {
	projectSlug := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)
	previewCfg := api.PreviewConfig{}
	ciContext := map[string]interface{}{}

	if err := files.Read(previewInitParamsPath, &previewCfg); err != nil {
		return err
	}

	if err := files.Read(previewDeployCiContextPath, &ciContext); err != nil {
		return err
	}

	env, err := deploy.Run(client, config.OrgID, projectSlug, &previewCfg, &ciContext)

	if err != nil {
		return err
	}

	var url = lipgloss.NewStyle().SetString(env.URL).Underline(true).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Deploying preview environment: %s", url)

	fmt.Println(msg)

	return nil
}

func runPreviewDecommission(cmd *cobra.Command, args []string) error {
	projectTargetSlugOrTargetID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	env, err := decommission.Run(client, config.OrgID, projectTargetSlugOrTargetID)

	if err != nil {
		return err
	}

	var url = lipgloss.NewStyle().SetString(env.URL).Underline(true).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Decommissioning preview environment: %s", url)
	fmt.Println(msg)

	return nil
}
