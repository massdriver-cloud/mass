package bundle

import (
	"fmt"
	"net/url"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/xeipuuv/gojsonschema"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunLint(b *bundle.Bundle, mdClient *client.Client) error {
	fmt.Println("Checking massdriver.yaml for errors...")

	greenCheckmark := prettylogs.Green(" ✓")

	bundleSchemaURL, err := url.JoinPath(mdClient.Config.URL, "json-schemas", "bundle.json")
	if err != nil {
		return fmt.Errorf("failed to construct bundle schema URL: %w", err)
	}
	schemaLoader := gojsonschema.NewReferenceLoader(bundleSchemaURL)
	err = b.LintSchema(schemaLoader)
	if err != nil {
		return err
	}
	fmt.Println(greenCheckmark, "Schema validation passed.")

	err = b.LintParamsConnectionsNameCollision()
	if err != nil {
		return err
	}
	fmt.Println(greenCheckmark, "Parameter and connection collision check passed.")

	err = b.LintMatchRequired()
	if err != nil {
		return err
	}
	fmt.Println(greenCheckmark, "Required parameters check passed.")

	err = b.LintInputsMatchProvisioner()
	if err != nil {
		return err
	}
	fmt.Println(greenCheckmark, "Inputs match provisioner check passed.")

	fmt.Println("Linting complete, massdriver.yaml is valid!")

	return nil
}
