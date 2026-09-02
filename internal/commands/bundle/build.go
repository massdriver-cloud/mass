// Package bundle provides command implementations for bundle operations.
package bundle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// RunBuild validates the bundle, then builds it at buildPath.
func RunBuild(buildPath string, b *bundle.Bundle, mdClient *massdriver.Client) error {
	if err := ValidateSchema(b, mdClient.Config().URL); err != nil {
		return err
	}
	return b.Build(buildPath, resourcetype.NewMassdriverResolver(mdClient))
}

// ValidateSchema must run before dereferencing, which assumes a schema-valid
// bundle. A fetch failure halts the build — the API owns the bundle format.
func ValidateSchema(b *bundle.Bundle, serverURL string) error {
	result := b.LintSchema(serverURL)
	if !result.HasErrors() {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("bundle failed schema validation:")
	for _, issue := range result.Errors() {
		fmt.Fprintf(&sb, "\n  - %s", issue.Message)
	}
	return errors.New(sb.String())
}
