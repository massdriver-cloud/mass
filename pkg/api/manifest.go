package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type Manifest struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Suffix      string `json:"suffix"`
	Description string `json:"description"`
}

func CreateManifest(ctx context.Context, mdClient *client.Client, bundleId string, projectId string, name string, slug string, description string) (*Manifest, error) {
	response, err := createManifest(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleId, projectId, name, slug, description)
	if err != nil {
		return nil, err
	}
	if !response.CreateManifest.Successful {
		messages := response.CreateManifest.GetMessages()
		if len(messages) > 0 {
			errMsg := "unable to create manifest:"
			for _, msg := range messages {
				errMsg += "\n  - " + msg.Message
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("unable to create manifest")
	}
	return toManifest(response.CreateManifest.Result)
}

func toManifest(v any) (*Manifest, error) {
	manifest := Manifest{}
	if err := mapstructure.Decode(v, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}
	return &manifest, nil
}
