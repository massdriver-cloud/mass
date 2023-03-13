package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
	peinit "github.com/massdriver-cloud/mass/internal/commands/preview_environments/initialize"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/spf13/cobra"
)

var previewCmdHelp = mustRenderHelpDoc("preview")
var previewInitCmdHelp = mustRenderHelpDoc("preview/init")
var previewDeployCmdHelp = mustRenderHelpDoc("preview/deploy")

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

	// cfg, err := commands.InitializePreviewEnvironment(client, c.OrgID, projectSlug, os.Stdin, os.Stdout)

	// if err != nil {
	// 	return err
	// }

	// return files.Write(previewInitParamsPath, cfg)

	// TODO: can we start the program here and return a model instead?!

	// TODO: dont call it run since it returns a model...
	m, _ := peinit.Run(client, c.OrgID, projectSlug)

	// TODO: can we return the m and let cobra start the command, we would need no
	// stdin / stdout mocks...
	// should all params be fields on the model?
	p := tea.NewProgram(m)
	result, err := p.Run()
	_ = err

	updatedModel, _ := (result).(peinit.Model)
	updatedModel.PrintSelections()

	return nil
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

	var url = lipgloss.NewStyle().SetString(env.URL).Underline(true).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Deploying preview environment: %s", url)

	fmt.Println(msg)

	return nil
}
