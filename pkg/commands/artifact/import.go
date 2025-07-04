package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/artifact"
	"github.com/massdriver-cloud/mass/pkg/jsonschema"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunImport(ctx context.Context, mdClient *client.Client, artifactName string, artifactType string, artifactFile string) (string, error) {
	bytes, readErr := os.ReadFile(artifactFile)
	if readErr != nil {
		return "", readErr
	}

	artifact := artifact.Artifact{}
	unmarshalErr := json.Unmarshal(bytes, &artifact)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}

	validateErr := validateArtifact(ctx, mdClient, artifactType, &artifact)
	if validateErr != nil {
		return "", validateErr
	}

	fmt.Printf("Creating artifact %s of type %s...\n", artifactName, artifactType)
	resp, createErr := api.CreateArtifact(ctx, mdClient, artifactName, artifactType, artifact.Data, artifact.Specs)
	if createErr != nil {
		return "", createErr
	}
	fmt.Printf("Artifact %s created! (Artifact ID: %s)\n", resp.Name, resp.ID)

	return resp.ID, nil
}

func validateArtifact(ctx context.Context, mdClient *client.Client, artifactType string, artifact *artifact.Artifact) error {
	ads, adsErr := api.ListArtifactDefinitions(ctx, mdClient)
	if adsErr != nil {
		return adsErr
	}

	var schemaMap map[string]any

	for _, ad := range ads {
		if ad.Name == artifactType {
			schemaMap = ad.Schema
		}
	}
	if schemaMap == nil {
		return fmt.Errorf("unable to find matching artifact definition: %s", artifactType)
	}

	sch, schemaErr := jsonschema.LoadSchemaFromGo(schemaMap)
	if schemaErr != nil {
		return fmt.Errorf("failed to compile artifact definition schema: %w", schemaErr)
	}
	return jsonschema.ValidateGo(sch, artifact)
}
