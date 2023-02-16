package cmd

import (
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var bundleCmdHelp = mustRenderHelpDoc("bundle")

var bundleTemplateCmdHelp = mustRenderHelpDoc("bundle/template")

var templateListCmdHelp = mustRenderHelpDoc("bundle/template-list")

var templateRefreshCmdHelp = mustRenderHelpDoc("bundle/template-refresh")

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Generate and publish bundles.",
	Long:  bundleCmdHelp,
}

var bundleTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Application template development tools",
	Long:  bundleTemplateCmdHelp,
}

var bundleTemplateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundle templates",
	Long:  templateListCmdHelp,
	RunE:  runBundleTemplateList,
}

var bundleTemplateRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Update template list from the official Massdriver Github",
	Long:  templateRefreshCmdHelp,
	RunE:  runBundleTemplateRefresh,
}

/*
var bundleNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a new bundle from a template",
	// TODO: Helpdoc
	RunE: runBundleNew,
}
*/

func init() {
	rootCmd.AddCommand(bundleCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	// bundleCmd.AddCommand(bundleNewCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateListCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateRefreshCmd)
}

func runBundleTemplateList(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	templateList, err := commands.ListTemplates(cache)
	// TODO: BubbleTea a nice data grid for this. Repo title row with template list sub rows.

	view := ""
	for _, repo := range templateList {
		templates := strings.Join(repo.Templates, "\n")
		view = fmt.Sprintf("Repository: %s\nTemplates:\n%s", repo.Repository, templates)
	}

	fmt.Println(view)
	return err
}

func runBundleTemplateRefresh(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	err := commands.RefreshTemplates(cache)

	return err
}

/*
func runBundleNew(cmd *cobra.Command, args []string) error {
	return nil
}
*/
