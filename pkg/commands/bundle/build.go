package bundle

import (
	"github.com/massdriver-cloud/mass/pkg/bundle"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunBuild(buildPath string, b *bundle.Bundle, mdClient *client.Client) error {
	return b.Build(buildPath, mdClient)
}
