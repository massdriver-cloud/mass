package publish

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"

	"oras.land/oras-go/v2/content/memory"
)

func Run(b *bundle.Bundle, mdClient *client.Client, buildFromDir string, tag string) error {
	ctx := context.Background()

	var printBundleName = prettylogs.Underline(b.Name)
	fmt.Printf("Publishing %s to package manager\n", printBundleName)

	repo, repoErr := getRepo(b, mdClient)
	if repoErr != nil {
		return fmt.Errorf("getting repository: %w", repoErr)
	}
	store := memory.New()
	publisher := &Publisher{
		Store: store,
		Repo:  repo,
	}

	fmt.Printf("Packaging bundle %s for package manager\n", printBundleName)

	manifestDescriptor, packageErr := publisher.PackageBundle(ctx, buildFromDir, tag)
	if packageErr != nil {
		return fmt.Errorf("packaging bundle: %w", packageErr)
	}

	fmt.Printf("Package %s created with digest: %s\n", printBundleName, manifestDescriptor.Digest)
	fmt.Printf("Pushing packaged bundle %s to package manager\n", printBundleName)

	publishErr := publisher.PublishBundle(ctx, tag)
	if publishErr != nil {
		return fmt.Errorf("publishing bundle: %w", publishErr)
	}

	fmt.Printf("Bundle %s successfully published\n", printBundleName)

	return nil
}
