// Package bundle provides command implementations for bundle operations.
package bundle

import (
	"github.com/massdriver-cloud/mass/pkg/bundle"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunBuild builds the bundle at buildPath using the provided bundle and client.
func RunBuild(buildPath string, b *bundle.Bundle, mdClient *client.Client) error {
	return b.Build(buildPath, mdClient)
}
