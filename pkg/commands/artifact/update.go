package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/pkg/api"
	artifactpkg "github.com/massdriver-cloud/mass/pkg/artifact"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunUpdate(ctx context.Context, mdClient *client.Client, artifactID string, artifactName string, artifactFile string) (string, error) {
	bytes, readErr := os.ReadFile(artifactFile)
	if readErr != nil {
		return "", readErr
	}

	payload := artifactpkg.Artifact{}
	unmarshalErr := json.Unmarshal(bytes, &payload)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}

	// Name is required by the backend. If not provided, fetch the existing artifact's name.
	if artifactName == "" {
		existing, getErr := api.GetArtifact(ctx, mdClient, artifactID)
		if getErr != nil {
			return "", fmt.Errorf("failed to get existing artifact: %w", getErr)
		}
		artifactName = existing.Name
	}

	fmt.Printf("Updating artifact %s...\n", artifactID)
	resp, updateErr := api.UpdateArtifact(ctx, mdClient, artifactID, artifactName, payload)
	if updateErr != nil {
		return "", updateErr
	}
	fmt.Printf("Artifact %s updated! (Artifact ID: %s)\n", resp.Name, resp.ID)

	return resp.ID, nil
}
