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
	"slices"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/cli"
	cmdbundle "github.com/massdriver-cloud/mass/internal/commands/bundle"
	"github.com/massdriver-cloud/mass/internal/params"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/templates"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/bundle.get.md.tmpl
var bundleTemplates embed.FS

type bundleNew struct {
	name         string
	description  string
	templateName string
	connections  []string
	outputDir    string
	paramsDir    string
}

type bundleList struct {
	search    string
	name      string
	sortField string
	sortOrder string
	output    string
}

// NewCmdBundle returns a cobra command for generating and publishing bundles.
func NewCmdBundle() *cobra.Command { //nolint:funlen // cobra command builders are necessarily long
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
	bundleListCmd.Flags().StringVarP(&bundleListInput.search, "search", "s", "", "Search bundles by name, readme, and changelog")
	bundleListCmd.Flags().StringVarP(&bundleListInput.name, "name", "n", "", "Filter by exact bundle name")
	bundleListCmd.Flags().StringVar(&bundleListInput.sortField, "sort", "", "Sort field (name, created_at). Defaults to name, or relevance when using --search")
	bundleListCmd.Flags().StringVar(&bundleListInput.sortOrder, "order", "asc", "Sort order (asc, desc)")
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
	return bundleCmd
}

func runBundleTemplateList(cmd *cobra.Command, args []string) error {
	templateList, err := templates.List()
	if err != nil {
		return err
	}

	if len(templateList) == 0 {
		fmt.Println("No templates found.")
		return nil
	}

	fmt.Println("Available templates:")
	for _, tmpl := range templateList {
		fmt.Printf("  %s\n", tmpl)
	}
	return nil
}

func runBundleNewInteractive(outputDir string, resourceTypeNames []string) (*templates.TemplateData, error) {
	templateData := &templates.TemplateData{
		OutputDir:     outputDir,
		ResourceTypes: resourceTypeNames,
	}

	err := bundle.RunPromptNew(templateData)
	if err != nil {
		return nil, err
	}

	return templateData, nil
}

func runBundleNewFlags(input *bundleNew) (*templates.TemplateData, error) {
	connectionData := make([]templates.Connection, len(input.connections))
	for i, conn := range input.connections {
		parts := strings.Split(conn, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid connection argument: %s", conn)
		}
		connectionData[i] = templates.Connection{
			ResourceType: parts[1],
			Name:         parts[0],
		}
	}

	templateData := &templates.TemplateData{
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

	var templateData *templates.TemplateData
	var runErr error
	if input.name == "" || input.templateName == "" {
		// run the interactive prompt
		mdClient, mdClientErr := client.New()
		if mdClientErr != nil {
			return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
		}

		resourceTypes, listErr := api.ListResourceTypes(ctx, mdClient, nil)
		if listErr != nil {
			return fmt.Errorf("error fetching resource types: %w", listErr)
		}
		resourceTypeNames := make([]string, len(resourceTypes))
		for i, rt := range resourceTypes {
			resourceTypeNames[i] = rt.ID
			if rt.Name != rt.ID {
				resourceTypeNames[i] += " (" + rt.Name + ")"
			}
		}
		slices.Sort(resourceTypeNames)

		templateData, runErr = runBundleNewInteractive(input.outputDir, resourceTypeNames)
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

	if newErr := cmdbundle.RunNew(templateData); newErr != nil {
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

	switch {
	case results.HasErrors():
		return fmt.Errorf("linting failed with %d error(s)", len(results.Errors()))
	case results.HasWarnings():
		fmt.Printf("Linting completed with %d warning(s)\n", len(results.Warnings()))
	default:
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

		switch {
		case results.HasErrors():
			fmt.Printf("Halting publish: Linting failed with %d error(s)\n", len(results.Errors()))
			os.Exit(1)
		case results.HasWarnings():
			if failWarnings {
				fmt.Printf("Halting publish: linting failed with %d warning(s)\n", len(results.Warnings()))
				os.Exit(1)
			}
			fmt.Printf("Linting completed with %d warning(s)\n", len(results.Warnings()))
		default:
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

	filter := &api.OciReposFilter{
		ArtifactType: "application/vnd.massdriver.bundle.v1+json",
		Search:       input.search,
	}
	if input.name != "" {
		filter.Name = &api.OciRepoNameFilter{Eq: input.name}
	}

	var sort *api.OciReposSort
	if input.sortField != "" {
		order := api.SortOrderAsc
		if strings.EqualFold(input.sortOrder, "desc") {
			order = api.SortOrderDesc
		}
		field := api.OciReposSortFieldName
		if strings.EqualFold(input.sortField, "created_at") {
			field = api.OciReposSortFieldCreatedAt
		}
		sort = &api.OciReposSort{Field: field, Order: order}
	}

	repos, err := api.ListOciRepos(ctx, mdClient, filter, sort)
	if err != nil {
		return fmt.Errorf("failed to list bundles: %w", err)
	}

	switch input.output {
	case "json":
		jsonBytes, err := json.MarshalIndent(repos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal bundles to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "table":
		tbl := cli.NewTable("Name", "Latest", "Created At")
		for _, repo := range repos {
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

	bundleID := args[0]
	if !strings.Contains(bundleID, "@") {
		bundleID = bundleID + "@latest"
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

	bundle, err := api.GetBundle(ctx, mdClient, bundleID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(bundle, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal bundle to JSON: %w", marshalErr)
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
	if renderErr := tmpl.Execute(&buf, data); renderErr != nil {
		return fmt.Errorf("failed to execute template: %w", renderErr)
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
