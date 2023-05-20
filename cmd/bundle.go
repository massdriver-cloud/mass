package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/commands/publish"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

var bundleNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new bundle from a template",
	RunE: func(cmd *cobra.Command, args []string) error {
		if name, _ := cmd.Flags().GetString("name"); name != "" {
			return runBundleNewFlags(cmd)
		} else {
			return runBundleNewInteractive(cmd)
		}
	},
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
	bundleNewCmd.Flags().StringP("name", "n", "", "Name of the new bundle")
	bundleNewCmd.Flags().StringP("description", "d", "", "Description of the new bundle")
	bundleNewCmd.Flags().StringP("template-type", "t", "", "Name of the bundle template to use")
	bundleNewCmd.Flags().StringSliceP("connections", "c", []string{}, "Connections and names to add to the bundle - example: massdriver/vpc=network")
	bundleNewCmd.Flags().StringP("output-directory", "o", ".", "Directory to output the new bundle")
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

func runBundleNewInteractive(cmd *cobra.Command) error {
	var (
		outputDir string
	)
	outputDir, _ = cmd.Flags().GetString("output-directory")

	// run the interactive prompt
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	err := commands.RefreshTemplates(cache)

	if err != nil {
		return err
	}

	templateData := &templatecache.TemplateData{
		Access: "private",
		// Promptui templates are a nightmare. Need to support multi repos when moving this to bubbletea
		TemplateRepo: "/massdriver-cloud/application-templates",
		// TODO: unify bundle build and app build outputDir logic and support
		OutputDir: outputDir,
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

func runBundleNewFlags(cmd *cobra.Command) error {
	var (
		name         string
		description  string
		templateName string
		connections  []string
		outputDir    string
	)

	name, _ = cmd.Flags().GetString("name")
	description, _ = cmd.Flags().GetString("description")
	templateName, _ = cmd.Flags().GetString("template-type")
	connections, _ = cmd.Flags().GetStringSlice("connections")
	outputDir, _ = cmd.Flags().GetString("output-directory")

	// parse flags
	if err := cmd.ParseFlags(nil); err != nil {
		return err
	}

	if name == "" || templateName == "" {
		return fmt.Errorf("name and template-type flags are required")
	}

	connectionData := make([]templatecache.Connection, len(connections))
	for i, conn := range connections {
		parts := strings.Split(conn, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid connection argument: %s", conn)
		}
		connectionData[i] = templatecache.Connection{
			ArtifactDefinition: parts[0],
			Name:               parts[1],
		}
	}

	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)

	templateData := &templatecache.TemplateData{
		Access:       "private",
		TemplateRepo: "/massdriver-cloud/application-templates",
		OutputDir:    outputDir,
		Name:         name,
		Description:  description,
		TemplateName: templateName,
		Connections:  connectionData,
	}

	err := commands.GenerateNewBundle(cache, templateData)
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

	unmarshalledBundle, err := unmarshalBundle(buildDirectory, cmd, fs)

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

	unmarshalledBundle, err := unmarshalBundle(buildDirectory, cmd, fs)

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

	unmarshalledBundle, err := unmarshalBundle(buildDirectory, cmd, fs)

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

func unmarshalBundle(readDirectory string, cmd *cobra.Command, fs afero.Fs) (*bundle.Bundle, error) {
	file, err := afero.ReadFile(fs, path.Join(readDirectory, "massdriver.yaml"))

	if err != nil {
		return nil, err
	}

	unmarshalledBundle := &bundle.Bundle{}

	err = yaml.Unmarshal(file, unmarshalledBundle)

	if err != nil {
		return nil, err
	}

	applyOverrides(unmarshalledBundle, cmd)

	if unmarshalledBundle.IsApplication() {
		applyAppBlockDefaults(unmarshalledBundle)
	}

	return unmarshalledBundle, nil
}

func applyAppBlockDefaults(b *bundle.Bundle) {
	if b.AppSpec != nil {
		if b.AppSpec.Envs == nil {
			b.AppSpec.Envs = map[string]string{}
		}
		if b.AppSpec.Policies == nil {
			b.AppSpec.Policies = []string{}
		}
		if b.AppSpec.Secrets == nil {
			b.AppSpec.Secrets = map[string]bundle.Secret{}
		}
	}
}

func applyOverrides(bundle *bundle.Bundle, cmd *cobra.Command) {
	access, err := cmd.Flags().GetString("access")

	if err == nil {
		bundle.Access = access
	}
}
