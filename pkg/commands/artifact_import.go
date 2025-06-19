package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/artifact"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/xeipuuv/gojsonschema"
)

func ArtifactImport(ctx context.Context, mdClient *client.Client, artifactName string, artifactType string, artifactFile string) (string, error) {
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

	var schema map[string]any

	for _, ad := range ads {
		if ad.Name == artifactType {
			schema = ad.Schema
		}
	}
	if schema == nil {
		return fmt.Errorf("invalid artifact type: %s", artifactType)
	}

	documentLoader := gojsonschema.NewGoLoader(artifact)
	schemaLoader := gojsonschema.NewGoLoader(schema)
	result, validateErr := gojsonschema.Validate(schemaLoader, documentLoader)
	if validateErr != nil {
		return validateErr
	}
	if !result.Valid() {
		errorString := "The artifact is not valid. see errors :\n"
		for _, desc := range result.Errors() {
			errorString += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf("%s", errorString)
	}
	return nil
}
