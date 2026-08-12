package resourcetype

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/internal/oci"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"oras.land/oras-go/v2/content/file"
)

// RunPull downloads a resource type from its OCI repository into directory,
// resolving version to a concrete tag. It returns the resolved tag and the
// pulled manifest digest.
func RunPull(ctx context.Context, mdClient *massdriver.Client, name, version, directory string) (string, string, error) {
	repo, repoErr := mdClient.OciRepos.Target(name)
	if repoErr != nil {
		return "", "", repoErr
	}

	tag, tagErr := resolveTag(ctx, mdClient, name, version)
	if tagErr != nil {
		return "", "", tagErr
	}

	store, fileErr := file.New(directory)
	if fileErr != nil {
		return "", "", fmt.Errorf("failed to create file store: %w", fileErr)
	}
	defer store.Close()

	puller := &oci.Puller{
		Target: store,
		Repo:   repo,
	}

	descriptor, pullErr := puller.Pull(ctx, tag)
	if pullErr != nil {
		return "", "", fmt.Errorf("failed to pull resource type: %w", pullErr)
	}

	return tag, descriptor.Digest.String(), nil
}

// resolveTag maps a user-supplied version (a concrete tag, a release channel
// name, or "latest") to a concrete OCI tag.
func resolveTag(ctx context.Context, mdClient *massdriver.Client, name, version string) (string, error) {
	repo, getErr := mdClient.OciRepos.Get(ctx, name)
	if getErr != nil {
		return "", fmt.Errorf("failed to get OCI repo: %w", getErr)
	}

	if version == "" || version == "latest" {
		// Prefer the "latest" release channel; otherwise fall back to the newest
		// tag (the Get query returns tags sorted by version, descending).
		if repo.LatestTag != "" {
			return repo.LatestTag, nil
		}
		if len(repo.Tags) > 0 {
			return repo.Tags[0].Tag, nil
		}
		return "", fmt.Errorf("no published versions found for resource type '%s'", name)
	}

	for _, t := range repo.Tags {
		if t.Tag == version {
			return version, nil
		}
	}

	for _, channel := range repo.ReleaseChannels {
		if version == channel.Name {
			return channel.Tag, nil
		}
	}

	return "", fmt.Errorf("version or release channel '%s' not found for resource type '%s'", version, name)
}
