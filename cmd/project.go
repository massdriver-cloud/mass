package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

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

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG")

	for _, project := range *projects {
		fmt.Printf("Project: %s\n", project.Name)
		line := fmt.Sprintf("%s\t%s\t%s", project.ID, project.Name, project.Slug)
		fmt.Fprintln(w, line)
	}

	w.Flush()

	// TODO: present UI
	// _, err := commands.DeployPackage(client, config.OrgID, name)

	return err
}
