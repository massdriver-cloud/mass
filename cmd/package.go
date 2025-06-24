package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/pkg"
	"github.com/massdriver-cloud/mass/pkg/files"

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

func NewCmdPkg() *cobra.Command {
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

	pkgConfigureCmd.Flags().StringVarP(&pkgParamsPath, "params", "p", pkgParamsPath, "Path to params JSON file. This file supports bash interpolation.")

	pkgDeployCmd := &cobra.Command{
		Use:     `deploy <project>-<env>-<manifest>`,
		Short:   "Deploy packages",
		Example: `mass package deploy ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("package/deploy"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgDeploy,
	}

	pkgDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

	pkgPatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual package parameter values",
		Example: `mass package patch ecomm-prod-db --set='.version = "13.4"'`,
		Long:    helpdocs.MustRender("package/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgPatch,
	}

	pkgPatchCmd.Flags().StringArrayVarP(&pkgPatchQueries, "set", "s", []string{}, "Sets a package parameter value using JQ expressions.")

	// pkg and infra are the same, lets reuse a get command/template here.
	pkgGetCmd := &cobra.Command{
		Use:     `get  <project>-<env>-<manifest>`,
		Short:   "Get a package",
		Aliases: []string{"g"},
		Example: `mass package get ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("package/get"),
		Args:    cobra.ExactArgs(1), // Enforce exactly one argument
		RunE:    runPkgGet,
	}
	pkgGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	pkgCmd.AddCommand(pkgDeployCmd)
	pkgCmd.AddCommand(pkgConfigureCmd)
	pkgCmd.AddCommand(pkgPatchCmd)
	pkgCmd.AddCommand(pkgGetCmd)

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

	pkg, err := api.GetPackageByName(ctx, mdClient, pkgID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(pkg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal package to JSON: %w", err)
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

	tmpl, err := template.New("package").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pkg); err != nil {
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
	if err := files.Read(pkgParamsPath, &params); err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err := pkg.RunConfigure(ctx, mdClient, packageSlugOrID, params)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Configuring: %s", name)
	fmt.Println(msg)

	return err
}

func runPkgPatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	packageSlugOrID := args[0]

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
