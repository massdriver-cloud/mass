package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/cobra"
)

var artifactCommandHelp = mustRenderHelpDoc("artifact")

var artifactCommand = &cobra.Command{
	Use:   "artifact",
	Short: "Manage Massdriver artifacts",
	Long:  artifactCommandHelp,
}

var listArtifactsCommand = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l", "ls"},
	Short:   "List all artifacts",
	RunE:    runListArtifacts,
}

var artifactInfoCommand = &cobra.Command{
	Use:     "info <artifact id>",
	Aliases: []string{"i"},
	Short:   "Get information about an artifact",
	Args:    cobra.ExactArgs(1),
	RunE:    runArtifactInfo,
}

func init() {
	rootCmd.AddCommand(artifactCommand)
	artifactCommand.AddCommand(listArtifactsCommand)
	artifactCommand.AddCommand(artifactInfoCommand)
}

func runListArtifacts(cmd *cobra.Command, args []string) error {
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)
	artifacts, err := api.GetAllArtifacts(client, c.OrgID)

	if err != nil {
		return err
	}

	fmt.Println("Artifacts:")
	for _, artifact := range artifacts {
		id := lipgloss.NewStyle().SetString(artifact.ID).Foreground(lipgloss.Color("#7D56F4"))
		kind := lipgloss.NewStyle().SetString(artifact.Type).Foreground(lipgloss.Color("#4026B0"))
		msg := fmt.Sprintf("%s: %s (%s)", artifact.Name, id, kind)
		fmt.Println(msg)
	}

	return nil
}

func runArtifactInfo(cmd *cobra.Command, args []string) error {
	c := config.Get()
	client := api.NewClient(c.URL, c.APIKey)
	artifact, err := api.GetArtifact(client, c.OrgID, args[0])

	if err != nil {
		return err
	}

	dataMarshalled, err := json.MarshalIndent(artifact.Data, "", "  ")

	if err != nil {
		return err
	}

	dataFormatted, err := glamour.Render("```json\n"+string(dataMarshalled)+"\n```", "dark")

	if err != nil {
		return err
	}

	name := lipgloss.NewStyle().SetString(artifact.Name).Foreground(lipgloss.Color("#7D56F4"))
	id := lipgloss.NewStyle().SetString(artifact.ID).Foreground(lipgloss.Color("#4026B0"))
	fmt.Printf("%s (%s)\n", name, id)
	fmt.Println(dataFormatted)

	return nil
}
