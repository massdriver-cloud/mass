package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/commands/publish"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/params"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/spf13/cobra"
)

// hiddenArtifacts are artifact definitions that the API returns that
// should not be added to bundles
var hiddenArtifacts = map[string]struct{}{
	"massdriver/api":        {},
	"massdriver/draft-node": {},
}

type bundleNew struct {
	name         string
	description  string
	templateName string
	connections  []string
	outputDir    string
	paramsDir    string
}

func NewCmdBundle() *cobra.Command {
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Generate and publish bundles",
		Long:  helpdocs.MustRender("bundle"),
	}

	bundleBuildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build schemas and generate IaC files from massdriver.yaml file",
		RunE:  runBundleBuild,
	}
	bundleBuildCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	bundleImportCmd := &cobra.Command{
		Use:          "import",
		Short:        "Import declared variables from IaC into massdriver.yaml params",
		Long:         helpdocs.MustRender("bundle/import"),
		RunE:         runBundleImport,
		SilenceUsage: true,
	}
	bundleImportCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
	bundleImportCmd.Flags().BoolP("all", "a", false, "Import all variables without prompting")

	bundleLintCmd := &cobra.Command{
		Use:          "lint",
		Short:        "Check massdriver.yaml file for common errors",
		SilenceUsage: true,
		RunE:         runBundleLint,
	}
	bundleLintCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	var bundleNewInput bundleNew

	bundleNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new bundle from a template",
		Long:  helpdocs.MustRender("bundle/new"),
		Run:   func(cmd *cobra.Command, args []string) { runBundleNew(&bundleNewInput) },
	}
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.name, "name", "n", "", "Name of the new bundle")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.description, "description", "d", "", "Description of the new bundle")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.templateName, "template-name", "t", "", "Name of the bundle template to use")
	bundleNewCmd.Flags().StringSliceVarP(&bundleNewInput.connections, "connections", "c", []string{}, "Connections and names to add to the bundle - example: network=massdriver/vpc")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.outputDir, "output-directory", "o", ".", "Directory to output the new bundle")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.paramsDir, "params-directory", "p", "", "Path with existing params to use - opentofu module directory or helm chart values.yaml")

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
	bundleCmd.AddCommand(bundleImportCmd)
	bundleCmd.AddCommand(bundleLintCmd)
	bundleCmd.AddCommand(bundleNewCmd)
	bundleCmd.AddCommand(bundlePublishCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateListCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateRefreshCmd)
	return bundleCmd
}

func runBundleTemplateList(cmd *cobra.Command, args []string) error {
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)
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
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)

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

func runBundleNewFlags(input *bundleNew) (*templatecache.TemplateData, error) {
	connectionData := make([]templatecache.Connection, len(input.connections))
	for i, conn := range input.connections {
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
		Access:             "private",
		TemplateRepo:       "/massdriver-cloud/application-templates",
		OutputDir:          input.outputDir,
		Name:               input.name,
		Description:        input.description,
		TemplateName:       input.templateName,
		Connections:        connectionData,
		ExistingParamsPath: input.paramsDir,
	}

	return templateData, nil
}

func runBundleNew(input *bundleNew) {
	cache, err := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)
	if err != nil {
		log.Fatal(err)
	}

	// If MD_TEMPLATES_PATH is set then it's most likely local dev work on templates so don't fetch
	// or the refresh will overwrite whatever path this points to
	if os.Getenv("MD_TEMPLATES_PATH") == "" {
		err = commands.RefreshTemplates(cache)
		if err != nil {
			log.Fatal(err)
		}
	}

	c, configErr := config.Get()
	if configErr != nil {
		log.Fatal(configErr)
	}
	gqlclient := api.NewClient(c.URL, c.APIKey)

	artifactDefs, err := api.GetArtifactDefinitions(gqlclient, c.OrgID)
	if err != nil {
		log.Fatal(err)
	}

	artifactDefinitions := map[string]map[string]interface{}{}
	for _, v := range artifactDefs {
		if _, ok := hiddenArtifacts[v.Name]; ok {
			continue
		}
		artifactDefinitions[v.Name] = v.Schema
	}

	bundle.SetMassdriverArtifactDefinitions(artifactDefinitions)

	var templateData *templatecache.TemplateData
	if input.name == "" || input.templateName == "" {
		// run the interactive prompt
		templateData, err = runBundleNewInteractive(input.outputDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// skip the interactive prompt and use flags
		templateData, err = runBundleNewFlags(input)
		if err != nil {
			log.Fatal(err)
		}
	}

	localParams, err := params.GetFromPath(templateData.TemplateName, templateData.ExistingParamsPath)
	if err != nil {
		log.Fatal(err)
	}

	templateData.ParamsSchema = localParams

	if err = commands.GenerateNewBundle(cache, templateData); err != nil {
		log.Fatal(err)
	}
}

func runBundleBuild(cmd *cobra.Command, args []string) error {
	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := bundle.UnmarshalAndApplyDefaults(buildDirectory)
	if err != nil {
		return err
	}

	c := restclient.NewClient()

	return commands.BuildBundle(buildDirectory, unmarshalledBundle, c)
}

func runBundleImport(cmd *cobra.Command, args []string) error {
	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	skipVerify, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}

	return commands.ImportParams(buildDirectory, skipVerify)
}

func runBundleLint(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := bundle.UnmarshalAndApplyDefaults(buildDirectory)
	if err != nil {
		return err
	}

	c := restclient.NewClient().WithAPIKey(config.APIKey)

	err = unmarshalledBundle.DereferenceSchemas(buildDirectory, c)
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

	buildDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}

	unmarshalledBundle, err := bundle.UnmarshalAndApplyDefaults(buildDirectory)
	if err != nil {
		return err
	}

	access, err := cmd.Flags().GetString("access")
	if err == nil {
		unmarshalledBundle.Access = access
	}

	c := restclient.NewClient().WithAPIKey(config.APIKey)

	err = commands.BuildBundle(buildDirectory, unmarshalledBundle, c)
	if err != nil {
		return err
	}

	return publish.Run(unmarshalledBundle, c, buildDirectory)
}
