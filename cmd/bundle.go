package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
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

// hiddenArtifacts are artifact definitions that the API returns that
// should not be added to bundles
var hiddenArtifacts = map[string]struct{}{
	"massdriver/api":        {},
	"massdriver/draft-node": {},
}

func NewCmdBundle() *cobra.Command {
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Generate and publish bundles",
		Long:  helpdocs.MustRender("bundle"),
	}

	bundleBuildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build schemas from massdriver.yaml file",
		RunE:  runBundleBuild,
	}
	bundleBuildCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	bundleLintCmd := &cobra.Command{
		Use:          "lint",
		Short:        "Check massdriver.yaml file for common errors",
		SilenceUsage: true,
		RunE:         runBundleLint,
	}
	bundleLintCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	bundleNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new bundle from a template",
		RunE:  runBundleNew,
	}

	bundlePublishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish bundle to Massdriver's package manager",
		RunE:  runBundlePublish,
	}
	bundlePublishCmd.Flags().String("access", "private", "Override the access, useful in CI for deploying to sandboxes.")
	bundlePublishCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	bundleTemplateCmd := &cobra.Command{
		Use:   "template",
		Short: "Application template development tools",
		Long:  helpdocs.MustRender("bundle/template"),
	}

	bundleTemplateListCmd := &cobra.Command{
		Use:   "list",
		Short: "List bundle templates",
		Long:  helpdocs.MustRender("bundle/template-list"),
		RunE:  runBundleTemplateList,
	}

	bundleTemplateRefreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Update template list from the official Massdriver Github",
		Long:  helpdocs.MustRender("bundle/template-refresh"),
		RunE:  runBundleTemplateRefresh,
	}

	bundleCmd.AddCommand(bundleBuildCmd)
	bundleCmd.AddCommand(bundleLintCmd)
	bundleCmd.AddCommand(bundleNewCmd)
	bundleCmd.AddCommand(bundlePublishCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateListCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateRefreshCmd)
	bundleNewCmd.Flags().StringP("name", "n", "", "Name of the new bundle")
	bundleNewCmd.Flags().StringP("description", "d", "", "Description of the new bundle")
	bundleNewCmd.Flags().StringP("template-type", "t", "", "Name of the bundle template to use")
	bundleNewCmd.Flags().StringSliceP("connections", "c", []string{}, "Connections and names to add to the bundle - example: massdriver/vpc=network")
	bundleNewCmd.Flags().StringP("output-directory", "o", ".", "Directory to output the new bundle")
	return bundleCmd
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

func runBundleNewInteractive(outputDir string) (*templatecache.TemplateData, error) {
	templateData := &templatecache.TemplateData{
		Access: "private",
		// Promptui templates are a nightmare. Need to support multi repos when moving this to bubbletea
		TemplateRepo: "/massdriver-cloud/application-templates",
		// TODO: unify bundle build and app build outputDir logic and support
		OutputDir: outputDir,
	}

	err := bundle.RunPromptNew(templateData)
	if err != nil {
		return nil, err
	}

	return templateData, nil
}

func runBundleNewFlags(cmd *cobra.Command) (*templatecache.TemplateData, error) {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return nil, err
	}

	connections, err := cmd.Flags().GetStringSlice("connections")
	if err != nil {
		return nil, err
	}

	templateName, err := cmd.Flags().GetString("template-type")
	if err != nil {
		return nil, err
	}

	outputDir, err := cmd.Flags().GetString("output-directory")
	if err != nil {
		return nil, err
	}

	connectionData := make([]templatecache.Connection, len(connections))
	for i, conn := range connections {
		parts := strings.Split(conn, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid connection argument: %s", conn)
		}
		connectionData[i] = templatecache.Connection{
			ArtifactDefinition: parts[1],
			Name:               parts[0],
		}
	}

	templateData := &templatecache.TemplateData{
		Access:       "private",
		TemplateRepo: "/massdriver-cloud/application-templates",
		OutputDir:    outputDir,
		Name:         name,
		Description:  description,
		TemplateName: templateName,
		Connections:  connectionData,
	}

	return templateData, nil
}

func runBundleNew(cmd *cobra.Command, args []string) error {
	fs := afero.NewOsFs()
	cache, err := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	if err != nil {
		return err
	}

	err = commands.RefreshTemplates(cache)
	if err != nil {
		return err
	}

	var (
		name         string
		templateName string
		outputDir    string
	)

	// define flag
	name, err = cmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	templateName, err = cmd.Flags().GetString("template-type")
	if err != nil {
		return err
	}

	outputDir, err = cmd.Flags().GetString("output-directory")
	if err != nil {
		return err
	}

	// parse flags
	if err = cmd.ParseFlags(args); err != nil {
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
		if _, ok := hiddenArtifacts[v.Name]; ok {
			continue
		}
		artifacts = append(artifacts, v.Name)
	}

	sort.StringSlice(artifacts).Sort()

	bundle.SetMassdriverArtifactDefinitions(artifacts)

	var templateData *templatecache.TemplateData
	if name == "" || templateName == "" {
		// run the interactive prompt
		templateData, err = runBundleNewInteractive(outputDir)
		if err != nil {
			return err
		}
	} else {
		// skip the interactive prompt and use flags
		templateData, err = runBundleNewFlags(cmd)
		if err != nil {
			return err
		}
	}

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

	// This looks weird but we have to be careful we don't overwrite things that do exist in the bundle file
	if unmarshalledBundle.Connections == nil {
		unmarshalledBundle.Connections = make(map[string]any)
	}

	if unmarshalledBundle.Connections["properties"] == nil {
		unmarshalledBundle.Connections["properties"] = make(map[string]any)
	}

	if unmarshalledBundle.Artifacts == nil {
		unmarshalledBundle.Artifacts = make(map[string]any)
	}

	if unmarshalledBundle.Artifacts["properties"] == nil {
		unmarshalledBundle.Artifacts["properties"] = make(map[string]any)
	}

	return unmarshalledBundle, nil
}

func applyOverrides(bundle *bundle.Bundle, cmd *cobra.Command) {
	access, err := cmd.Flags().GetString("access")
	if err == nil {
		bundle.Access = access
	}
}
