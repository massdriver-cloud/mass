package bundle

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	sdkbundle "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/bundle"

	"oras.land/oras-go/v2/content/memory"
)

// RunPublish packages and publishes a bundle to the Massdriver registry.
func RunPublish(ctx context.Context, b *bundle.Bundle, mdClient *client.Client, buildFromDir string, developmentRelease bool) error {
	version, versionErr := getVersion(ctx, mdClient, b, developmentRelease)
	if versionErr != nil {
		return versionErr
	}

	var printBundleName = prettylogs.Underline(b.Name)
	var printOrganizationID = prettylogs.Underline(mdClient.Config.OrganizationID)
	fmt.Printf("Publishing %s:%s to organization %s...\n", printBundleName, version, printOrganizationID)

	repo, repoErr := sdkbundle.GetBundleRepository(mdClient, b.Name)
	if repoErr != nil {
		return fmt.Errorf("getting repository: %w", repoErr)
	}
	store := memory.New()
	publisher := &bundle.Publisher{
		Store: store,
		Repo:  repo,
	}

	fmt.Printf("Packaging bundle %s...\n", printBundleName)

	manifestDescriptor, packageErr := publisher.PackageBundle(ctx, buildFromDir, version)
	if packageErr != nil {
		return fmt.Errorf("packaging bundle: %w", packageErr)
	}

	fmt.Printf("Package %s created with digest: %s\n", printBundleName, manifestDescriptor.Digest)
	fmt.Printf("Pushing %s to package manager...\n", printBundleName)

	publishErr := publisher.PublishBundle(ctx, version)
	if publishErr != nil {
		return fmt.Errorf("publishing bundle: %w", publishErr)
	}

	fmt.Printf("Bundle %s:%s successfully published to organization %s!\n", printBundleName, version, printOrganizationID)

	// Output repo instances URL
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		instancesURL := urlHelper.RepoInstancesURL(b.Name, version)
		fmt.Printf("🔗 %s\n", instancesURL)
	}

	return nil
}

func getVersion(ctx context.Context, mdClient *client.Client, b *bundle.Bundle, developmentRelease bool) (string, error) {
	existingVersions, err := api.GetOciRepoTags(ctx, mdClient, b.Name)
	if err != nil {
		// TODO need actual error handling here. Need to distinguish between "repo not found" and other errors
		existingVersions = []string{}
	}

	if b.Version != "0.0.0" && slices.Contains(existingVersions, b.Version) {
		if !developmentRelease {
			return "", fmt.Errorf("version %s already exists for bundle %s", b.Version, b.Name)
		}
		return "", fmt.Errorf("version %s already exists for bundle %s - cannot publish a development release for an existing version", b.Version, b.Name)
	}

	version := b.Version
	if developmentRelease {
		timestamp := time.Now().UTC().Format("20060102T150405Z")
		version = fmt.Sprintf("%s-dev.%s", b.Version, timestamp)
	}
	return version, nil
}
