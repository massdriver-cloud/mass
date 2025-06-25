package definition

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/definition"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunPublish(ctx context.Context, mdClient *client.Client, in io.Reader) error {
	artDefBytes, readErr := io.ReadAll(in)
	if readErr != nil {
		return fmt.Errorf("failed to read artifact definition: %w", readErr)
	}

	artDefMap := make(map[string]any)
	if err := json.Unmarshal(artDefBytes, &artDefMap); err != nil {
		return fmt.Errorf("failed to unmarshal artifact definition: %w", err)
	}

	validateErr := definition.Validate(mdClient, artDefBytes)
	if validateErr != nil {
		return validateErr
	}

	_, publishErr := api.PublishArtifactDefinition(ctx, mdClient, artDefMap)
	return publishErr
}
