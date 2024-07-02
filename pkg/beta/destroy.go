package beta

import (
	"context"

	"github.com/docker/docker/client"
)

func Destroy(ctx context.Context, cli *client.Client, name string, params map[string]interface{}, connections map[string]interface{}) error {
	return executeStep(ctx, cli, PROVISIONER_ACTION_DESTROY, name, params, connections)
}
