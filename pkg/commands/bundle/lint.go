package bundle

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunLint(b *bundle.Bundle, mdClient *client.Client) bundle.LintResult {
	fmt.Println("Checking massdriver.yaml for errors...")

	var allResults bundle.LintResult

	// Schema validation
	schemaResult := b.LintSchema(mdClient)
	allResults.Merge(schemaResult)
	printLintResult("Schema validation", schemaResult)

	// Parameter and connection collision check
	collisionResult := b.LintParamsConnectionsNameCollision()
	allResults.Merge(collisionResult)
	printLintResult("Parameter and connection collision", collisionResult)

	// Required parameters check
	requiredResult := b.LintMatchRequired()
	allResults.Merge(requiredResult)
	printLintResult("Required parameters", requiredResult)

	// Inputs match provisioner check
	inputsResult := b.LintInputsMatchProvisioner()
	allResults.Merge(inputsResult)
	printLintResult("Inputs match provisioner", inputsResult)

	return allResults
}

func printLintResult(ruleName string, result bundle.LintResult) {
	greenCheckmark := prettylogs.Green(" ✓")
	orangeWarning := prettylogs.Orange(" !")
	redError := prettylogs.Red(" ✗")

	if result.HasErrors() {
		fmt.Printf("%s %s check failed with errors: \n", redError, ruleName)
		for _, issue := range result.Issues {
			fmt.Println(issue)
		}
	} else if result.HasWarnings() {
		fmt.Printf("%s %s check completed with warnings: \n", orangeWarning, ruleName)
		for _, warning := range result.Warnings() {
			fmt.Println(warning)
		}
	} else {
		fmt.Printf("%s %s check passed.\n", greenCheckmark, ruleName)
	}
}
