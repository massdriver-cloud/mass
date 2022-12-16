package cmd

import (
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/template_cache"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var bundleCmdHelp = mustRender(`
# Generate and publish Massdriver bundles.
`)

var bundleTemplateCmdHelp = mustRender(`
_This_ is where the help file goes
`)

var templateListCmdHelp = mustRender(`
_This_ is where the help file goes
`)

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

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundle templates",
	Long:  templateListCmdHelp,
	RunE:  runTemplateList,
}

var templateRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Update template list from the official Massdriver Github",
	Long:  templateListCmdHelp,
	RunE:  runTemplateRefresh,
}

func init() {
	rootCmd.AddCommand(bundleCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	bundleTemplateCmd.AddCommand(templateListCmd)
	bundleTemplateCmd.AddCommand(templateRefreshCmd)
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	var fs afero.Fs = afero.NewOsFs()
	cache, _ := template_cache.NewBundleTemplateCache(template_cache.GithubTemplatesFetcher, fs)
	templateList, err := commands.ListTemplates(cache)
	fmt.Printf("Application templates:\n  %s\n", strings.Join(templateList, "\n  "))
	return err
}

func runTemplateRefresh(cmd *cobra.Command, args []string) error {
	var fs afero.Fs = afero.NewOsFs()
	cache, _ := template_cache.NewBundleTemplateCache(template_cache.GithubTemplatesFetcher, fs)
	err := commands.RefreshTemplates(cache)

	return err
}
