package resourcetype

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
)

// RunDelete removes a resource type's OCI repository, refusing locally when it
// already has tags. Prompting is the caller's responsibility.
func RunDelete(ctx context.Context, mdClient *massdriver.Client, name string) (*ocirepos.OciRepo, error) {
	repo, getErr := mdClient.OciRepos.Get(ctx, name)
	if getErr != nil {
		return nil, fmt.Errorf("fetching OCI repo: %w", getErr)
	}

	if len(repo.Tags) > 0 {
		return nil, fmt.Errorf("resource type %s has published versions and is immutable; its repository cannot be deleted", name)
	}

	return mdClient.OciRepos.Delete(ctx, name)
}
