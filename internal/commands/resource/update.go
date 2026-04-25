package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/internal/api"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunUpdate updates an existing resource with the data from the given file.
func RunUpdate(ctx context.Context, mdClient *client.Client, resourceID string, resourceName string, resourceFile string) (string, error) {
	bytes, readErr := os.ReadFile(resourceFile)
	if readErr != nil {
		return "", readErr
	}

	var payload map[string]any
	unmarshalErr := json.Unmarshal(bytes, &payload)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}

	// Name is required by the backend. If not provided, fetch the existing resource's name.
	if resourceName == "" {
		existing, getErr := api.GetResource(ctx, mdClient, resourceID)
		if getErr != nil {
			return "", fmt.Errorf("failed to get existing resource: %w", getErr)
		}
		resourceName = existing.Name
	}

	input := api.UpdateResourceInput{
		Name:    resourceName,
		Payload: payload,
	}

	fmt.Printf("Updating resource %s...\n", resourceID)
	resp, updateErr := api.UpdateResource(ctx, mdClient, resourceID, input)
	if updateErr != nil {
		return "", updateErr
	}
	fmt.Printf("Resource %s updated! (Resource ID: %s)\n", resp.Name, resp.ID)

	return resp.ID, nil
}
