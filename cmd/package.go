package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/spf13/cobra"
)

var pkgCmdHelp = helpdocs.MustRender("package")
var pkgGetCmdHelp = helpdocs.MustRender("package/get")

// TODO: support common READ outputs: table, json, 'show'

var pkgCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"pkg"},
	Short:   "Manage deployed packages",
	Long:    pkgCmdHelp,
}

var pkgGetCmd = &cobra.Command{
	Use:     `get`,
	Short:   "Get a package",
	Aliases: []string{"g"},
	Long:    pkgGetCmdHelp,
	Args:    cobra.ExactArgs(1), // Enforce exactly one argument
	RunE:    runPkgGet,
}

// Bundle: foo
// ActiveDeployment: nil or id
// Params: pretty print?
// Env: name

func init() {
	rootCmd.AddCommand(pkgCmd)
	pkgCmd.AddCommand(pkgGetCmd)
}

func runPkgGet(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	client := api.NewClient(config.URL, config.APIKey)
	pkgId := args[0]

	pkg, err := api.GetPackageByName(client, config.OrgID, pkgId)

	if err != nil {
		return err
	}

	renderPackage(pkg)

	return nil
}

func renderPackage(pkg *api.Package) error {
	paramsJSON, err := json.MarshalIndent(pkg.Params, "", "  ")
	if err != nil {
		return err
	}

	md := fmt.Sprintf(`# Package Summary

**Package:** %s

**Bundle:** %s

**Environment:** %s

## Parameters
`+"```json"+`
%s
`+"```"+`
`, pkg.NamePrefix, pkg.Manifest.Bundle.Name, pkg.Environment.Slug, string(paramsJSON))

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(md)
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}
