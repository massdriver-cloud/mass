package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/config"
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
