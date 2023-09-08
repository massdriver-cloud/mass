package commands

import (
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/artifact"
	"github.com/spf13/afero"
	"github.com/xeipuuv/gojsonschema"
)

func ArtifactImport(client graphql.Client, orgID string, fs afero.Fs, artifactName string, artifactType string, artifactFile string) (string, error) {
	bytes, readErr := afero.ReadFile(fs, artifactFile)
	if readErr != nil {
		return "", readErr
	}

	artifact := artifact.Artifact{}
	unmarshalErr := json.Unmarshal(bytes, &artifact)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}

	validateErr := validateArtifact(client, orgID, artifactType, &artifact)
	if validateErr != nil {
		return "", validateErr
	}

	fmt.Printf("Creating artifact %s of type %s...\n", artifactName, artifactType)
	resp, createErr := api.CreateArtifact(client, orgID, artifactName, artifactType, artifact.Data, artifact.Specs)
	if createErr != nil {
		return "", createErr
	}
	fmt.Printf("Artifact %s created! (Artifact ID: %s)\n", resp.Name, resp.ID)

	return resp.ID, nil
}

func validateArtifact(client graphql.Client, orgID string, artifactType string, artifact *artifact.Artifact) error {
	ads, adsErr := api.GetArtifactDefinitions(client, orgID)
	if adsErr != nil {
		return adsErr
	}

	var schema map[string]interface{}

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
