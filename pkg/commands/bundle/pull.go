package bundle

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	sdkbundle "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/bundle"
	"oras.land/oras-go/v2/content/file"
)

func RunPull(ctx context.Context, mdClient *client.Client, bundleName string, version string, directory string) error {

	fmt.Printf("Pulling bundle %s:%s from organization %s to directory %s\n",
		prettylogs.Underline(bundleName),
		prettylogs.Underline(version),
		prettylogs.Underline(mdClient.Config.OrganizationID),
		prettylogs.Underline(directory),
	)

	repo, repoErr := sdkbundle.GetBundleRepository(mdClient, bundleName)
	if repoErr != nil {
		return repoErr
	}

	tag, tagErr := resolveTag(ctx, mdClient, bundleName, version)
	if tagErr != nil {
		return tagErr
	}

	store, fileErr := file.New(directory)
	if fileErr != nil {
		return fmt.Errorf("failed to create file store: %w", fileErr)
	}
	defer store.Close()

	puller := &bundle.Puller{
		Target: store,
		Repo:   repo,
	}

	descriptor, pullErr := puller.PullBundle(ctx, tag)
	if pullErr != nil {
		return fmt.Errorf("failed to pull bundle: %w", pullErr)
	}

	fmt.Printf("Bundle %s:%s pulled successfully (Digest: %s)\n",
		prettylogs.Underline(bundleName),
		prettylogs.Underline(tag),
		prettylogs.Underline(descriptor.Digest.String()),
	)

	return nil
}

func resolveTag(ctx context.Context, mdClient *client.Client, bundleName string, version string) (string, error) {
	repo, getErr := api.GetOciRepo(ctx, mdClient, bundleName)
	if getErr != nil {
		return "", fmt.Errorf("failed to get OCI repo: %w", getErr)
	}

	for _, tag := range repo.Tags {
		if version == tag.Tag {
			return version, nil
		}
	}

	for _, channel := range repo.ReleaseChannels {
		if version == channel.Name {
			return channel.Tag, nil
		}
	}

	return "", fmt.Errorf("version or release channel '%s' not found in OCI repo '%s'", version, bundleName)
}
