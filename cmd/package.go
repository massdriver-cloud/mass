package cmd

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/pkg"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/massdriver-cloud/mass/internal/prettylogs"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

var (
	pkgParamsPath   = "./params.json"
	pkgPatchQueries []string
)

//go:embed templates/package.get.md.tmpl
var packageTemplates embed.FS

// NewCmdPkg returns a cobra command for managing packages of IaC deployed in environments.
func NewCmdPkg() *cobra.Command { //nolint:funlen // cobra command builders are necessarily long
	pkgCmd := &cobra.Command{
		Use:     "package",
		Aliases: []string{"pkg"},
		Short:   "Manage packages of IaC deployed in environments.",
		Long:    helpdocs.MustRender("package"),
	}

	pkgConfigureCmd := &cobra.Command{
		Use:     `configure <project>-<env>-<manifest>`,
		Short:   "Configure package",
		Aliases: []string{"cfg"},
		Example: `mass package configure ecomm-prod-vpc --params=params.json`,
		Long:    helpdocs.MustRender("package/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgConfigure,
	}
	pkgConfigureCmd.Flags().StringVarP(&pkgParamsPath, "params", "p", pkgParamsPath, "Path to params json, tfvars or yaml file. Use '-' to read from stdin. This file supports bash interpolation.")

	pkgDeployCmd := &cobra.Command{
		Use:     `deploy <project>-<env>-<manifest>`,
		Short:   "Deploy packages",
		Example: `mass package deploy ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("package/deploy"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgDeploy,
	}
	pkgDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	pkgExportCmd := &cobra.Command{
		Use:     `export <project>-<env>-<manifest>`,
		Short:   "Export packages",
		Example: `mass package export ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("package/export"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgExport,
	}

	// pkg and infra are the same, lets reuse a get command/template here.
	pkgGetCmd := &cobra.Command{
		Use:     `get  <project>-<env>-<manifest>`,
		Short:   "Get a package",
		Aliases: []string{"g"},
		Example: `mass package get ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("package/get"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgGet,
	}
	pkgGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	pkgPatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Example: `mass package patch ecomm-prod-db --set='.version = "13.4"'`,
		Long:    helpdocs.MustRender("package/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgPatch,
	}
	pkgPatchCmd.Flags().StringArrayVarP(&pkgPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	pkgCreateCmd := &cobra.Command{
		Use:     `create [slug]`,
		Short:   "Create a manifest (add bundle to project)",
		Example: `mass package create dbbundle-test-serverless --bundle aws-rds-cluster`,
		Long:    helpdocs.MustRender("package/create"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgCreate,
	}
	pkgCreateCmd.Flags().StringP("name", "n", "", "Manifest name (defaults to slug if not provided)")
	pkgCreateCmd.Flags().StringP("bundle", "b", "", "Bundle ID or name (required)")
	_ = pkgCreateCmd.MarkFlagRequired("bundle")

	pkgVersionCmd := &cobra.Command{
		Use:     `version <package-id>@<version>`,
		Short:   "Set package version",
		Example: `mass package version api-prod-db@latest --release-channel development`,
		Long:    helpdocs.MustRender("package/version"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgVersion,
	}
	pkgVersionCmd.Flags().String("release-channel", "stable", "Release strategy (stable or development)")

	pkgDestroyCmd := &cobra.Command{
		Use:     `destroy <project>-<env>-<manifest>`,
		Short:   "Destroy (decommission) a package",
		Example: `mass package destroy api-prod-db --force`,
		Long:    "Destroy (decommission) a package. This will permanently delete the package and all its resources.",
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgDestroy,
	}
	pkgDestroyCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	pkgResetCmd := &cobra.Command{
		Use:     `reset <project>-<env>-<manifest>`,
		Short:   "Reset package status to 'Initialized'",
		Example: `mass package reset api-prod-db`,
		Long:    helpdocs.MustRender("package/reset"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgReset,
	}
	pkgResetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	pkgListCmd := &cobra.Command{
		Use:     `list <project>-<env>`,
		Short:   "List packages in an environment",
		Aliases: []string{"ls"},
		Example: `mass package list ecomm-prod`,
		Long:    helpdocs.MustRender("package/list"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgList,
	}

	pkgCmd.AddCommand(pkgConfigureCmd)
	pkgCmd.AddCommand(pkgDeployCmd)
	pkgCmd.AddCommand(pkgExportCmd)
	pkgCmd.AddCommand(pkgGetCmd)
	pkgCmd.AddCommand(pkgListCmd)
	pkgCmd.AddCommand(pkgPatchCmd)
	pkgCmd.AddCommand(pkgCreateCmd)
	pkgCmd.AddCommand(pkgVersionCmd)
	pkgCmd.AddCommand(pkgDestroyCmd)
	pkgCmd.AddCommand(pkgResetCmd)

	return pkgCmd
}

func runPkgGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	pkgID := args[0]

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	pkg, err := api.GetPackage(ctx, mdClient, pkgID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(pkg, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal package to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderPackage(pkg)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func renderPackage(pkg *api.Package) error {
	tmplBytes, err := packageTemplates.ReadFile("templates/package.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	funcMap := template.FuncMap{
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}
	tmpl, err := template.New("package").Funcs(funcMap).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, pkg); renderErr != nil {
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

func runPkgDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]

	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err = pkg.RunDeploy(ctx, mdClient, name, msg)

	return err
}

func runPkgConfigure(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]

	params := map[string]any{}
	if pkgParamsPath == "-" {
		// Read from stdin
		if err := json.NewDecoder(os.Stdin).Decode(&params); err != nil {
			return fmt.Errorf("failed to decode JSON from stdin: %w", err)
		}
	} else {
		if err := files.Read(pkgParamsPath, &params); err != nil {
			return err
		}
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	configuredPkg, err := pkg.RunConfigure(ctx, mdClient, packageSlugOrID, params)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Package `%s` configured successfully\n", configuredPkg.Slug)

	// Get package details to build URL
	pkgDetails, err := api.GetPackage(ctx, mdClient, configuredPkg.Slug)
	if err == nil && pkgDetails.Environment != nil && pkgDetails.Environment.Project != nil && pkgDetails.Manifest != nil {
		urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.PackageURL(pkgDetails.Environment.Project.Slug, pkgDetails.Environment.Slug, pkgDetails.Manifest.Slug))
		}
	}

	return nil
}

func runPkgPatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err := pkg.RunPatch(ctx, mdClient, packageSlugOrID, pkgPatchQueries)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Patching: %s", name)
	fmt.Println(msg)

	return err
}

func runPkgExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	exportErr := pkg.RunExport(ctx, mdClient, packageSlugOrID)
	if exportErr != nil {
		return fmt.Errorf("failed to export package: %w", exportErr)
	}

	return nil
}

func runPkgCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fullSlug := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	bundleIDOrName, err := cmd.Flags().GetString("bundle")
	if err != nil {
		return err
	}

	// Parse project-env-manifest format: extract project (first), env (second), and manifest (third)
	// Format is $proj-$env-$manifest where each part has no hyphens
	parts := strings.Split(fullSlug, "-")
	if len(parts) != 3 {
		return fmt.Errorf("unable to determine project, environment, and manifest from slug %s (expected format: project-env-manifest)", fullSlug)
	}
	projectIDOrSlug := parts[0]
	environmentSlug := parts[1]
	manifestSlug := parts[2]

	if name == "" {
		name = manifestSlug
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err = api.CreateManifest(ctx, mdClient, bundleIDOrName, projectIDOrSlug, name, manifestSlug, "")
	if err != nil {
		return err
	}

	fmt.Printf("✅ Package `%s` created successfully\n", fullSlug)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.PackageURL(projectIDOrSlug, environmentSlug, manifestSlug))
	}
	return nil
}

func runPkgVersion(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageIDAndVersion := args[0]

	// Parse package-id@version format
	parts := strings.Split(packageIDAndVersion, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected <package-id>@<version>, got %s", packageIDAndVersion)
	}
	packageID := parts[0]
	version := parts[1]

	releaseChannel, err := cmd.Flags().GetString("release-channel")
	if err != nil {
		return err
	}

	// Convert release channel to ReleaseStrategy enum value
	var releaseStrategy api.ReleaseStrategy
	switch releaseChannel {
	case "development":
		releaseStrategy = api.ReleaseStrategyDevelopment
	case "stable":
		releaseStrategy = api.ReleaseStrategyStable
	default:
		return fmt.Errorf("invalid release-channel: must be 'stable' or 'development', got '%s'", releaseChannel)
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	updatedPkg, err := api.SetPackageVersion(ctx, mdClient, packageID, version, releaseStrategy)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Package `%s` version set successfully\n", updatedPkg.Slug)

	// Get package details to build URL
	pkgDetails, err := api.GetPackage(ctx, mdClient, updatedPkg.Slug)
	if err == nil && pkgDetails.Environment != nil && pkgDetails.Environment.Project != nil && pkgDetails.Manifest != nil {
		urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.PackageURL(pkgDetails.Environment.Project.Slug, pkgDetails.Environment.Slug, pkgDetails.Manifest.Slug))
		}
	}

	return nil
}

func runPkgDestroy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get package details for confirmation and URL
	pkg, err := api.GetPackage(ctx, mdClient, packageSlugOrID)
	if err != nil {
		return err
	}

	// Prompt for confirmation - requires typing the package slug unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently decommission package `%s` and all its resources.\n", pkg.Slug)
		fmt.Printf("Type `%s` to confirm decommission: ", pkg.Slug)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != pkg.Slug {
			fmt.Println("Decommission cancelled.")
			return nil
		}
	}

	_, err = api.DecommissionPackage(ctx, mdClient, pkg.ID, "")
	if err != nil {
		return err
	}

	fmt.Printf("✅ Package `%s` decommission started\n", pkg.Slug)

	// Get package details to build URL
	if pkg.Environment != nil && pkg.Environment.Project != nil && pkg.Manifest != nil {
		urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.PackageURL(pkg.Environment.Project.Slug, pkg.Environment.Slug, pkg.Manifest.Slug))
		}
	}

	return nil
}

func runPkgReset(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get package details for confirmation
	pkgDetails, err := api.GetPackage(ctx, mdClient, packageSlugOrID)
	if err != nil {
		return err
	}

	// Prompt for confirmation unless --force is used
	if !force {
		fmt.Printf("%s: This will reset package `%s` to 'Initialized' state and delete deployment history.\n", prettylogs.Orange("WARNING"), pkgDetails.Slug)
		fmt.Printf("Type `%s` to confirm reset: ", pkgDetails.Slug)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != pkgDetails.Slug {
			fmt.Println("Reset cancelled.")
			return nil
		}
	}

	pkg, err := pkg.RunReset(ctx, mdClient, packageSlugOrID)
	if err != nil {
		return err
	}

	var name = lipgloss.NewStyle().SetString(pkg.Slug).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("✅ Package %s reset successfully", name)
	fmt.Println(msg)

	return nil
}

func runPkgList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentSlug := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	env, err := api.GetEnvironment(ctx, mdClient, environmentSlug)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Bundle", "Status")

	for _, p := range env.Packages {
		name := ""
		if p.Manifest != nil {
			name = p.Manifest.Name
		}
		bundleName := ""
		if p.Bundle != nil {
			bundleName = p.Bundle.Name
		}
		tbl.AddRow(p.Slug, name, bundleName, p.Status)
	}

	tbl.Print()

	return nil
}
