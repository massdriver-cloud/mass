package resourcetype

import (
	"context"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// Delete removes a resource type by name. UX (confirmation prompt, success
// message) is the caller's responsibility — see [cmd.runTypeDelete].
func Delete(ctx context.Context, mdClient *massdriver.Client, name string) (*ResourceType, error) {
	return api.DeleteResourceType(ctx, mdClient, name)
}
