package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/cobra"
)

var projCmdHelp = mustRenderHelpDoc("project")
var projListCmdHelp = mustRenderHelpDoc("project/list")
var projCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"prj"},
	Short:   "Manage Projects",
	Long:    projCmdHelp,
}

var projListCmd = &cobra.Command{
	Use:     `list`,
	Short:   "List projects",
	Aliases: []string{"ls"},
	Long:    projListCmdHelp,
	RunE:    runProjList,
}

func init() {
	rootCmd.AddCommand(projCmd)
	projCmd.AddCommand(projListCmd)
}

func runProjList(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	client := api.NewClient(config.URL, config.APIKey)

	projects, err := api.ListProjects(client, config.OrgID)

	for _, project := range *projects {
		fmt.Printf("Project: %s\n", project.Name)
	}

	// TODO: present UI
	// _, err := commands.DeployPackage(client, config.OrgID, name)

	return err
}
