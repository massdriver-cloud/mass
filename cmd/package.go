package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/commands/package/configure"
	"github.com/massdriver-cloud/mass/pkg/commands/package/patch"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/massdriver-cloud/mass/pkg/files"
	"github.com/spf13/cobra"
)

func runPkgGet(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	client := api.NewClient(config.URL, config.APIKey)
	pkgID := args[0]

	pkg, err := api.GetPackageByName(client, config.OrgID, pkgID)

	if err != nil {
		return err
	}

	err = renderPackage(pkg)

	return err
}

func renderPackage(pkg *api.Package) error {
	paramsJSON, err := json.MarshalIndent(pkg.Params, "", "  ")
	if err != nil {
		return err
	}

	md := fmt.Sprintf(`# Package: %s

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

func runPkgDeploy(cmd *cobra.Command, args []string) error {
	name := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}

	_, err = commands.DeployPackage(client, config.OrgID, name, msg)

	return err
}

func runPkgConfigure(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)
	params := map[string]interface{}{}
	if err := files.Read(appParamsPath, &params); err != nil {
		return err
	}

	_, err := configure.Run(client, config.OrgID, packageSlugOrID, params)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Configuring: %s", name)
	fmt.Println(msg)

	return err
}

func runPkgPatch(cmd *cobra.Command, args []string) error {
	packageSlugOrID := args[0]
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	client := api.NewClient(config.URL, config.APIKey)

	_, err := patch.Run(client, config.OrgID, packageSlugOrID, appPatchQueries)

	var name = lipgloss.NewStyle().SetString(packageSlugOrID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Patching: %s", name)
	fmt.Println(msg)

	return err
}
