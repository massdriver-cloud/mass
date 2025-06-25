package definition

import (
	"context"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/definition"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunGet(ctx context.Context, mdClient *client.Client, definitionName string) (*api.ArtifactDefinitionWithSchema, error) {
	return definition.Get(ctx, mdClient, definitionName)
}
