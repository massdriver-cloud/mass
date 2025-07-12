package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/massdriver-cloud/airlock/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	cmdbundle "github.com/massdriver-cloud/mass/pkg/commands/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands/bundle/templates"
	"github.com/massdriver-cloud/mass/pkg/params"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
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
		Use:   "import",
		Short: "Import declared variables from IaC into massdriver.yaml params",
		Long:  helpdocs.MustRender("bundle/import"),
		RunE:  runBundleImport,
	}
	bundleImportCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
	bundleImportCmd.Flags().BoolP("all", "a", false, "Import all variables without prompting")

	bundleLintCmd := &cobra.Command{
		Use:   "lint",
		Short: "Check massdriver.yaml file for common errors",
		RunE:  runBundleLint,
	}
	bundleLintCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")

	var bundleNewInput bundleNew

	bundleNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new bundle from a template",
		Long:  helpdocs.MustRender("bundle/new"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runBundleNew(&bundleNewInput)
		},
	}
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.name, "name", "n", "", "Name of the new bundle. Setting this along with --template-name will disable the interactive prompt.")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.description, "description", "d", "", "Description of the new bundle")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.templateName, "template-name", "t", "", "Name of the bundle template to use. Setting this along with --name will disable the interactive prompt.")
	bundleNewCmd.Flags().StringSliceVarP(&bundleNewInput.connections, "connections", "c", []string{}, "Connections and names to add to the bundle - example: network=massdriver/vpc")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.outputDir, "output-directory", "o", ".", "Directory to output the new bundle")
	bundleNewCmd.Flags().StringVarP(&bundleNewInput.paramsDir, "params-directory", "p", "", "Path with existing params to use - opentofu module directory or helm chart values.yaml")

	bundlePublishCmd := &cobra.Command{
		Use:     "publish",
		Aliases: []string{"push"},
		Short:   "Publish bundle to Massdriver's package manager",
		RunE:    runBundlePublish,
	}
	bundlePublishCmd.Flags().StringP("build-directory", "b", ".", "Path to a directory containing a massdriver.yaml file.")
	bundlePublishCmd.Flags().String("access", "", "(Deprecated) Only here for backwards compatibility. Will be removed in a future release.")

	bundlePullCmd := &cobra.Command{
		Use:   "pull <bundle-name>",
		Short: "Pull bundle from Massdriver to local directory",
		Args:  cobra.ExactArgs(1),
		RunE:  runBundlePull,
	}
	bundlePullCmd.Flags().StringP("directory", "d", "", "Directory to output the bundle. Defaults to bundle name.")
	bundlePullCmd.Flags().BoolP("force", "f", false, "Force pull even if the directory already exists. This will overwrite existing files.")
	bundlePullCmd.Flags().StringP("tag", "t", "latest", "Bundle tag (defaults to 'latest')")

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
	bundleCmd.AddCommand(bundlePullCmd)
	bundleCmd.AddCommand(bundleTemplateCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateListCmd)
	bundleTemplateCmd.AddCommand(bundleTemplateRefreshCmd)
	return bundleCmd
}

func runBundleTemplateList(cmd *cobra.Command, args []string) error {
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)
	templateList, err := templates.RunList(cache)
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

	return templates.RunRefresh(cache)
}

func runBundleNewInteractive(outputDir string) (*templatecache.TemplateData, error) {
	templateData := &templatecache.TemplateData{
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

func runBundleNew(input *bundleNew) error {
	ctx := context.Background()

	cache, cacheErr := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)
	if cacheErr != nil {
		return fmt.Errorf("error initializing template cache: %w", cacheErr)
	}

	// If MD_TEMPLATES_PATH is set then it's most likely local dev work on templates so don't fetch
	// or the refresh will overwrite whatever path this points to
	if os.Getenv("MD_TEMPLATES_PATH") == "" {
		refreshErr := templates.RunRefresh(cache)
		if refreshErr != nil {
			return fmt.Errorf("error refreshing template cache: %w", refreshErr)
		}
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	artifactDefs, listErr := api.ListArtifactDefinitions(ctx, mdClient)
	if listErr != nil {
		return fmt.Errorf("error listing artifact definitions: %w", listErr)
	}

	artifactDefinitions := map[string]map[string]any{}
	for _, v := range artifactDefs {
		if _, ok := hiddenArtifacts[v.Name]; ok {
			continue
		}
		artifactDefinitions[v.Name] = v.Schema
	}

	bundle.SetMassdriverArtifactDefinitions(artifactDefinitions)

	var templateData *templatecache.TemplateData
	var runErr error
	if input.name == "" || input.templateName == "" {
		// run the interactive prompt
		templateData, runErr = runBundleNewInteractive(input.outputDir)
		if runErr != nil {
			return fmt.Errorf("error running interactive prompt: %w", runErr)
		}
	} else {
		// skip the interactive prompt and use flags
		templateData, runErr = runBundleNewFlags(input)
		if runErr != nil {
			return fmt.Errorf("error running flags: %w", runErr)
		}
	}

	localParams, paramsErr := params.GetFromPath(templateData.TemplateName, templateData.ExistingParamsPath)
	if paramsErr != nil {
		return fmt.Errorf("error getting params from path: %w", paramsErr)
	}
	templateData.ParamsSchema = localParams

	if newErr := cmdbundle.RunNew(cache, templateData); newErr != nil {
		return fmt.Errorf("error running bundle new: %w", newErr)
	}

	fmt.Printf("Bundle %q created successfully at path %q\n", templateData.Name, templateData.OutputDir)
	return nil
}

func runBundleBuild(cmd *cobra.Command, args []string) error {
	bundleDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	unmarshalledBundle, err := bundle.Unmarshal(bundleDirectory)
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	return cmdbundle.RunBuild(bundleDirectory, unmarshalledBundle, mdClient)
}

func runBundleImport(cmd *cobra.Command, args []string) error {
	bundleDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	skipVerify, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	return cmdbundle.RunImport(bundleDirectory, skipVerify)
}

func runBundleLint(cmd *cobra.Command, args []string) error {
	bundleDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	unmarshalledBundle, err := bundle.Unmarshal(bundleDirectory)
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	err = unmarshalledBundle.DereferenceSchemas(bundleDirectory, mdClient)
	if err != nil {
		return err
	}

	return cmdbundle.RunLint(unmarshalledBundle, mdClient)
}

func runBundlePublish(cmd *cobra.Command, args []string) error {
	access, _ := cmd.Flags().GetString("access")
	if access != "" {
		prettylogs.Orange("Warning: The --access flag is deprecated and will be removed in a future release.")
		fmt.Println(prettylogs.Orange("Warning: The --access flag is deprecated and will be removed in a future release."))
	}
	bundleDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	tag := "latest"
	cmd.SilenceUsage = true

	unmarshalledBundle, err := bundle.Unmarshal(bundleDirectory)
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	err = unmarshalledBundle.Build(bundleDirectory, mdClient)
	if err != nil {
		return err
	}

	return cmdbundle.RunPublish(unmarshalledBundle, mdClient, bundleDirectory, tag)
}

func runBundlePull(cmd *cobra.Command, args []string) error {
	bundleName := args[0]
	directory, _ := cmd.Flags().GetString("directory")
	if directory == "" {
		directory = bundleName
	}
	force, _ := cmd.Flags().GetBool("force")
	tag, _ := cmd.Flags().GetString("tag")
	cmd.SilenceUsage = true

	// Check if bundle exists in the specified directory and if so prompt the user
	mdYamlPath := filepath.Join(directory, "massdriver.yaml")
	if _, err := os.Stat(mdYamlPath); err == nil && !force {
		fmt.Printf("Bundle already exists at %s. Continuing will overwrite its contents. Continue? (y/N): ", mdYamlPath)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Bundle pull aborted!")
			return nil
		}
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return mdClientErr
	}

	pullErr := cmdbundle.RunPull(mdClient, bundleName, tag, directory)
	if pullErr != nil {
		return fmt.Errorf("error pulling bundle: %w", pullErr)
	}

	return nil
}
