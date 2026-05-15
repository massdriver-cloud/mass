// Package bundle provides command implementations for bundle operations.
package bundle

import (
	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// RunBuild builds the bundle at buildPath using the provided bundle and client.
func RunBuild(buildPath string, b *bundle.Bundle, mdClient *massdriver.Client) error {
	return b.Build(buildPath, resourcetype.NewMassdriverResolver(mdClient))
}
