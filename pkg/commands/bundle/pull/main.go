package pull

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	sdkbundle "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/bundle"
	"oras.land/oras-go/v2/content/file"
)

func Run(mdClient *client.Client, bundleName string, tag string, directory string) error {
	ctx := context.Background()

	prettyBundleName := prettylogs.Underline(bundleName)
	prettyOrganizationId := prettylogs.Underline(mdClient.Config.OrganizationID)
	prettyDirectory := prettylogs.Underline(directory)
	fmt.Printf("Pulling bundle %s from organization %s to directory %s\n", prettyBundleName, prettyOrganizationId, prettyDirectory)

	repo, repoErr := sdkbundle.GetBundleRepository(mdClient, bundleName)
	if repoErr != nil {
		return repoErr
	}

	store, fileErr := file.New(directory)
	if fileErr != nil {
		return fileErr
	}
	defer store.Close()

	puller := &Puller{
		Target: store,
		Repo:   repo,
	}

	descriptor, pullErr := puller.PullBundle(ctx, tag)
	if pullErr != nil {
		return pullErr
	}
	prettyDigest := prettylogs.Underline(descriptor.Digest.String())

	fmt.Printf("Bundle %s pulled successfully (Digest: %s)\n", prettyBundleName, prettyDigest)
	return nil
}
