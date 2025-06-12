package cmd

import (
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/rodaine/table"
)

var projCmdHelp = helpdocs.MustRender("project")
var projListCmdHelp = helpdocs.MustRender("project/list")
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

	headerFmt := color.New(color.FgHiBlue, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgHiWhite).SprintfFunc()

	tbl := table.New("ID", "Name", "Monthly $", "Daily $")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, project := range projects {
		tbl.AddRow(project.Slug, project.Name, project.MonthlyAverageCost, project.DailyAverageCost)
	}

	tbl.Print()

	return err
}
