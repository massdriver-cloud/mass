package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/commands/publish"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var bundleCmdHelp = mustRenderHelpDoc("bundle")

var bundleTemplateCmdHelp = mustRenderHelpDoc("bundle/template")

var templateListCmdHelp = mustRenderHelpDoc("bundle/template-list")

var templateRefreshCmdHelp = mustRenderHelpDoc("bundle/template-refresh")

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Generate and publish bundles",
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

var bundleNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new bundle from a template",
	RunE:  runBundleNew,
}

var bundleBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build schemas from massdriver.yaml file",
	RunE:  runBundleBuild,
}

var bundleLintCmd = &cobra.Command{
	Use:          "lint",
	Short:        "Check massdriver.yaml file for common errors",
	SilenceUsage: true,
	RunE:         runBundleLint,
}

var bundlePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish bundle to Massdriver's package manager",
	RunE:  runBundlePublish,
}

func init() {
	rootCmd.AddCommand(bundleCmd)
	bundleCmd.AddCommand(bundleNewCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateListCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateRefreshCmd)
	bundleCmd.AddCommand(bundleBuildCmd)
	bundleBuildCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
	bundleCmd.AddCommand(bundleLintCmd)
	bundleLintCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
	bundleCmd.AddCommand(bundlePublishCmd)
	bundlePublishCmd.Flags().String("access", "private", "Override the access, useful in CI for deploying to sandboxes.")
	bundlePublishCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
}

func runBundleTemplateList(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	templateList, err := commands.ListTemplates(cache)
	if err != nil {
		return err
	}
	// TODO: BubbleTea a nice data grid for this. Repo title row with template list sub rows.

	view := ""
	for _, repo := range templateList {
		templates := strings.Join(repo.Templates, "\n")
		view = fmt.Sprintf("Repository: %s\nTemplates:\n%s", repo.Repository, templates)
	}

	fmt.Println(view)
	return nil
}

func runBundleTemplateRefresh(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)

	return commands.RefreshTemplates(cache)
}

func runBundleNew(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	err := commands.RefreshTemplates(cache)
	if err != nil {
		return err
	}

	c, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	gqlclient := api.NewClient(c.URL, c.APIKey)

	artifactDefs, err := api.GetArtifactDefinitions(gqlclient, c.OrgID)
	if err != nil {
		return err
	}

	var artifacts []string
	for _, v := range artifactDefs {
		artifacts = append(artifacts, v.Name)
	}

	sort.StringSlice(artifacts).Sort()

	bundle.SetMassdriverArtifactDefinitions(artifacts)

	templateData := &templatecache.TemplateData{
		Access: "private",
		// Promptui templates are a nightmare. Need to support multi repos when moving this to bubbletea
		TemplateRepo: "/massdriver-cloud/application-templates",
		// TODO: unify bundle build and app build outputDir logic and support
		OutputDir: ".",
	}

	err = bundle.RunPromptNew(templateData)
	if err != nil {
		return err
	}

	err = commands.GenerateNewBundle(cache, templateData)
	if err != nil {
		return err
	}

	return nil
}

func runBundleBuild(cmd *cobra.Command, args []string) error {
	var fs = afero.NewOsFs()

	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := unmarshalBundleandApplyDefaults(buildDirectory, cmd, fs)
	if err != nil {
		return err
	}

	c := restclient.NewClient()

	err = commands.BuildBundle(buildDirectory, unmarshalledBundle, c, fs)

	return err
}

func runBundleLint(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	var fs = afero.NewOsFs()

	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := unmarshalBundleandApplyDefaults(buildDirectory, cmd, fs)
	if err != nil {
		return err
	}

	c := restclient.NewClient().WithAPIKey(config.APIKey)

	err = unmarshalledBundle.DereferenceSchemas(buildDirectory, c, fs)
	if err != nil {
		return err
	}

	return commands.LintBundle(unmarshalledBundle)
}

func runBundlePublish(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	var fs = afero.NewOsFs()

	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := unmarshalBundleandApplyDefaults(buildDirectory, cmd, fs)
	if err != nil {
		return err
	}

	c := restclient.NewClient().WithAPIKey(config.APIKey)

	err = commands.BuildBundle(buildDirectory, unmarshalledBundle, c, fs)
	if err != nil {
		return err
	}

	return publish.Run(unmarshalledBundle, c, fs, buildDirectory)
}

func unmarshalBundleandApplyDefaults(readDirectory string, cmd *cobra.Command, fs afero.Fs) (*bundle.Bundle, error) {
	unmarshalledBundle, err := bundle.UnmarshalBundle(readDirectory, fs)
	if err != nil {
		return nil, err
	}

	applyOverrides(unmarshalledBundle, cmd)

	if unmarshalledBundle.IsApplication() {
		bundle.ApplyAppBlockDefaults(unmarshalledBundle)
	}

	return unmarshalledBundle, nil
}

func applyOverrides(bundle *bundle.Bundle, cmd *cobra.Command) {
	access, err := cmd.Flags().GetString("access")
	if err == nil {
		bundle.Access = access
	}
}
