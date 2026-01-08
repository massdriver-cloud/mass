package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/preview"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

var (
	previewInitParamsPath      = "./preview.json"
	previewDeployCiContextPath = "/home/runner/work/_temp/_github_workflow/event.json"
)

func NewCmdPreview() *cobra.Command {
	previewCmd := &cobra.Command{
		Use:     "preview",
		Aliases: []string{"pv"},
		Short:   "Create and deploy preview environments",
		Long:    helpdocs.MustRender("preview"),
	}

	previewInitCmd := &cobra.Command{
		Use:   `init $projectSlug`,
		Short: "Generate preview environment configuration",
		Long:  helpdocs.MustRender("preview/init"),
		Args:  cobra.ExactArgs(1),
		RunE:  runPreviewInit,
		Example: `  # Initialize preview configuration
  mass preview init myproject

  # Specify custom output path
  mass preview init myproject --output ./preview-config.json`,
	}
	previewInitCmd.Flags().StringVarP(&previewInitParamsPath, "output", "o", "./preview.json", "Output path for preview environment params file. This file supports bash interpolation and can be manually edited or programatically modified during CI.")

	previewDeployCmd := &cobra.Command{
		Use:     "deploy",
		Aliases: []string{"apply"},
		Short:   "Deploy preview environment",
		Long:    helpdocs.MustRender("preview/deploy"),
		RunE:    runPreviewDeploy,
		Example: `  # Deploy using default preview.json
  mass preview deploy

  # Deploy with custom params file
  mass preview deploy --params ./custom-preview.json`,
	}
	previewDeployCmd.Flags().StringVarP(&previewInitParamsPath, "params", "p", previewInitParamsPath, "Path to preview environment configuration file. This file supports bash interpolation.")
	previewDeployCmd.Flags().StringVarP(&previewDeployCiContextPath, "ci-context", "c", previewDeployCiContextPath, "Path to GitHub Actions event.json")

	previewDecommissionCmd := &cobra.Command{
		Use:     "decommission $projectTargetSlug",
		Aliases: []string{"destroy"},
		Short:   "Decommission preview environment",
		Long:    helpdocs.MustRender("preview/decommission"),
		RunE:    runPreviewDecommission,
		Args:    cobra.ExactArgs(1),
		Example: `  # Decommission a preview environment
  mass preview decommission myproject-pr-123`,
	}

	previewCmd.AddCommand(previewInitCmd)
	previewCmd.AddCommand(previewDeployCmd)
	previewCmd.AddCommand(previewDecommissionCmd)

	return previewCmd
}

func runPreviewInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectSlug := args[0]

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	initModel, err := preview.RunNew(ctx, mdClient, projectSlug)
	if err != nil {
		return err
	}
	p := tea.NewProgram(initModel)
	result, err := p.Run()

	if err != nil {
		return err
	}

	updatedModel, _ := (result).(preview.Model)
	cfg := updatedModel.PreviewConfig()

	return files.Write(previewInitParamsPath, cfg)
}

func runPreviewDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	previewCfg := api.PreviewConfig{}
	ciContext := map[string]any{}

	if err := files.Read(previewInitParamsPath, &previewCfg); err != nil {
		return err
	}

	if err := files.Read(previewDeployCiContextPath, &ciContext); err != nil {
		return err
	}

	env, err := preview.RunDeploy(ctx, mdClient, previewCfg.ProjectSlug, &previewCfg, &ciContext)

	if err != nil {
		return err
	}

	var url = lipgloss.NewStyle().SetString(env.URL(ctx, mdClient)).Underline(true).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Deploying preview environment: %s", url)

	fmt.Println(msg)

	return nil
}

func runPreviewDecommission(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectTargetSlugOrTargetID := args[0]

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	env, err := preview.RunDecommission(ctx, mdClient, projectTargetSlugOrTargetID)

	if err != nil {
		return err
	}

	var url = lipgloss.NewStyle().SetString(env.URL(ctx, mdClient)).Underline(true).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Decommissioning preview environment: %s", url)
	fmt.Println(msg)

	return nil
}
