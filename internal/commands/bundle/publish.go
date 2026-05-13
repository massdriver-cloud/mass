package bundle

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"

	"oras.land/oras-go/v2/content/memory"
)

// RunPublish packages and publishes a bundle to the Massdriver registry.
func RunPublish(ctx context.Context, b *bundle.Bundle, mdClient *massdriver.Client, buildFromDir string, developmentRelease bool) error {
	version, err := getVersion(ctx, mdClient, b, developmentRelease)
	if err != nil {
		return err
	}

	cfg := mdClient.Config()
	var printBundleName = prettylogs.Underline(b.Name)
	var printOrganizationID = prettylogs.Underline(cfg.OrganizationID)
	fmt.Printf("Publishing %s:%s to organization %s...\n", printBundleName, version, printOrganizationID)

	repo, repoErr := mdClient.OciRepos.Target(b.Name)
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
	instancesURL := mdClient.URLs.Helper(ctx).RepoInstancesURL(b.Name, version)
	fmt.Printf("🔗 %s\n", instancesURL)

	return nil
}

// getVersion fetches the repo's existing tags and delegates to
// resolveVersion for the actual rule.
func getVersion(ctx context.Context, mdClient *massdriver.Client, b *bundle.Bundle, developmentRelease bool) (string, error) {
	repo, err := mdClient.OciRepos.Get(ctx, b.Name)
	if err != nil {
		return "", fmt.Errorf("fetching OCI repo: %w", err)
	}
	return resolveVersion(b, repo.Tags, developmentRelease)
}

// resolveVersion returns the tag to publish under, refusing to overwrite an
// existing non-development tag.
func resolveVersion(b *bundle.Bundle, existingTags []string, developmentRelease bool) (string, error) {
	if b.Version != "0.0.0" && slices.Contains(existingTags, b.Version) {
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
