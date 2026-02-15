package cmd

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/cli"
	cmdbundle "github.com/massdriver-cloud/mass/pkg/commands/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands/bundle/templates"
	"github.com/massdriver-cloud/mass/pkg/params"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/bundle.get.md.tmpl
var bundleTemplates embed.FS

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
	artifacts    []string
	outputDir    string
	paramsDir    string
}

type bundleList struct {
	search    string
	sortField string
	sortOrder string
	limit     int
	output    string
}

func NewCmdBundle() *cobra.Command {
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Generate and publish bundles",
		Long:  helpdocs.MustRender("bundle"),
	}

	var bundleListInput bundleList

	bundleListCmd := &cobra.Command{
		Use:   "list",
		Short: "List bundles in your organization",
		Long:  helpdocs.MustRender("bundle/list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runBundleList(&bundleListInput)
		},
	}
	bundleListCmd.Flags().StringVarP(&bundleListInput.search, "search", "s", "", "Search bundles (supports AND, OR, -, quotes)")
	bundleListCmd.Flags().StringVar(&bundleListInput.sortField, "sort", "name", "Sort field (name, created_at)")
	bundleListCmd.Flags().StringVar(&bundleListInput.sortOrder, "order", "asc", "Sort order (asc, desc)")
	bundleListCmd.Flags().IntVarP(&bundleListInput.limit, "limit", "l", 0, "Maximum number of results to return")
	bundleListCmd.Flags().StringVarP(&bundleListInput.output, "output", "o", "table", "Output format (table, json)")

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
	bundleNewCmd.Flags().StringSliceVarP(&bundleNewInput.artifacts, "artifacts", "a", []string{}, "Artifacts and names to add to the bundle - example: network=massdriver/vpc")
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
	bundlePublishCmd.Flags().BoolP("development", "d", false, "Publish the bundle as a development release.")
	bundlePublishCmd.Flags().String("access", "", "(Deprecated) Only here for backwards compatibility. Will be removed in a future release.")
	bundlePublishCmd.Flags().BoolP("fail-warnings", "f", false, "Fail on warnings from the linter")
	bundlePublishCmd.Flags().BoolP("skip-lint", "s", false, "Skip linting")

	bundleGetCmd := &cobra.Command{
		Use:   "get <bundle-name>[@<version>]",
		Short: "Get bundle information from Massdriver",
		Long:  helpdocs.MustRender("bundle/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runBundleGet,
	}
	bundleGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	bundlePullCmd := &cobra.Command{
		Use:   "pull <bundle-name>",
		Short: "Pull bundle from Massdriver to local directory",
		Args:  cobra.ExactArgs(1),
		RunE:  runBundlePull,
	}
	bundlePullCmd.Flags().StringP("directory", "d", "", "Directory to output the bundle. Defaults to bundle name.")
	bundlePullCmd.Flags().BoolP("force", "f", false, "Force pull even if the directory already exists. This will overwrite existing files.")
	bundlePullCmd.Flags().StringP("version", "v", "latest", "Bundle version or release channel")

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

	bundleCmd.AddCommand(bundleListCmd)
	bundleCmd.AddCommand(bundleBuildCmd)
	bundleCmd.AddCommand(bundleImportCmd)
	bundleCmd.AddCommand(bundleLintCmd)
	bundleCmd.AddCommand(bundleNewCmd)
	bundleCmd.AddCommand(bundlePublishCmd)
	bundleCmd.AddCommand(bundleGetCmd)
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

	artifactData := make([]templatecache.Artifact, len(input.artifacts))
	for i, art := range input.artifacts {
		parts := strings.Split(art, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid connection argument: %s", art)
		}
		artifactData[i] = templatecache.Artifact{
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
		Artifacts:          artifactData,
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
	if paramsErr == nil {
		templateData.ParamsSchema = localParams
	}

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

	results := cmdbundle.RunLint(unmarshalledBundle, mdClient)

	if results.HasErrors() {
		return fmt.Errorf("linting failed with %d error(s)", len(results.Errors()))
	} else if results.HasWarnings() {
		fmt.Printf("Linting completed with %d warning(s)\n", len(results.Warnings()))
	} else {
		fmt.Println("Linting completed, massdriver.yaml is valid!")
	}

	return nil
}

func runBundlePublish(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	access, _ := cmd.Flags().GetString("access")
	if access != "" {
		fmt.Println(prettylogs.Orange("Warning: The --access flag is deprecated and will be removed in a future release."))
	}
	bundleDirectory, err := cmd.Flags().GetString("build-directory")
	if err != nil {
		return err
	}
	failWarnings, err := cmd.Flags().GetBool("fail-warnings")
	if err != nil {
		return err
	}
	skipLint, err := cmd.Flags().GetBool("skip-lint")
	if err != nil {
		return err
	}

	developmentRelease, err := cmd.Flags().GetBool("development")
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

	err = unmarshalledBundle.Build(bundleDirectory, mdClient)
	if err != nil {
		return err
	}

	if !skipLint {
		results := cmdbundle.RunLint(unmarshalledBundle, mdClient)

		if results.HasErrors() {
			fmt.Printf("Halting publish: Linting failed with %d error(s)\n", len(results.Errors()))
			os.Exit(1)
		} else if results.HasWarnings() {
			if failWarnings {
				fmt.Printf("Halting publish: linting failed with %d warning(s)\n", len(results.Warnings()))
				os.Exit(1)
			}
			fmt.Printf("Linting completed with %d warning(s)\n", len(results.Warnings()))
		} else {
			fmt.Println("Linting completed, massdriver.yaml is valid!")
		}
	}

	return cmdbundle.RunPublish(ctx, unmarshalledBundle, mdClient, bundleDirectory, developmentRelease)
}

func runBundlePull(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	bundleName := args[0]
	directory, _ := cmd.Flags().GetString("directory")
	if directory == "" {
		directory = bundleName
	}
	force, _ := cmd.Flags().GetBool("force")
	version, _ := cmd.Flags().GetString("version")
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

	pullErr := cmdbundle.RunPull(ctx, mdClient, bundleName, version, directory)
	if pullErr != nil {
		return fmt.Errorf("error pulling bundle: %w", pullErr)
	}

	return nil
}

func runBundleList(input *bundleList) error {
	ctx := context.Background()

	mdClient, err := client.New()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	opts := api.ReposListOptions{
		Search:    input.search,
		SortField: input.sortField,
		SortOrder: input.sortOrder,
		Limit:     input.limit,
	}

	page, err := api.ListRepos(ctx, mdClient, opts)
	if err != nil {
		return fmt.Errorf("failed to list bundles: %w", err)
	}

	switch input.output {
	case "json":
		jsonBytes, err := json.MarshalIndent(page, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal bundles to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "table":
		tbl := cli.NewTable("Name", "Latest", "Created At")
		for _, repo := range page.Items {
			latest := ""
			for _, rc := range repo.ReleaseChannels {
				if rc.Name == "latest" {
					latest = rc.Tag
					break
				}
			}
			createdAt := repo.CreatedAt.Format("2006-01-02 15:04:05")
			tbl.AddRow(repo.Name, latest, createdAt)
		}
		tbl.Print()
	default:
		return fmt.Errorf("unsupported output format: %s", input.output)
	}

	return nil
}

func runBundleGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	arg := args[0]
	parts := strings.Split(arg, "@")
	bundleId := parts[0]
	version := "latest"
	if len(parts) == 2 {
		version = parts[1]
	}

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	bundle, err := api.GetBundle(ctx, mdClient, bundleId, &version)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(bundle, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal bundle to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderBundle(bundle, mdClient)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func renderBundle(b *api.Bundle, mdClient *client.Client) error {
	tmplBytes, err := bundleTemplates.ReadFile("templates/bundle.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("bundle").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Get app URL for constructing bundle URL
	ctx := context.Background()
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	bundleURL := ""
	if urlErr == nil {
		bundleURL = urlHelper.BundleURL(b.Name, b.Version)
	}

	data := struct {
		*api.Bundle
		URL string
	}{
		Bundle: b,
		URL:    bundleURL,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(buf.String())
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}
