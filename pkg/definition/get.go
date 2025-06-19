package definition

import (
	"context"
	"encoding/json"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func Get(ctx context.Context, mdClient *client.Client, definitionName string) (*api.ArtifactDefinitionWithSchema, error) {
	return api.GetArtifactDefinition(ctx, mdClient, definitionName)
}

func GetAsMap(ctx context.Context, mdClient *client.Client, definitionName string) (map[string]any, error) {
	ad, getErr := Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return nil, getErr
	}

	adData, marshallErr := json.Marshal(ad)
	if marshallErr != nil {
		return nil, marshallErr
	}

	var result map[string]interface{}
	unmarshalErr := json.Unmarshal(adData, &result)
	return result, unmarshalErr
}
