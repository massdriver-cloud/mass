package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// Manifest represents a deployed bundle instance (package) within a project environment.
type Manifest struct {
	ID          string `json:"id" mapstructure:"id"`
	Slug        string `json:"slug" mapstructure:"slug"`
	Name        string `json:"name" mapstructure:"name"`
	Suffix      string `json:"suffix" mapstructure:"suffix"`
	Description string `json:"description" mapstructure:"description"`
}

// CreateManifest creates a new manifest (package) in the specified project in the Massdriver API.
func CreateManifest(ctx context.Context, mdClient *client.Client, bundleID string, projectID string, name string, slug string, description string) (*Manifest, error) {
	response, err := createManifest(ctx, mdClient.GQL, mdClient.Config.OrganizationID, bundleID, projectID, name, slug, description)
	if err != nil {
		return nil, err
	}
	if !response.CreateManifest.Successful {
		messages := response.CreateManifest.GetMessages()
		if len(messages) > 0 {
			var sb strings.Builder
			sb.WriteString("unable to create manifest:")
			for _, msg := range messages {
				sb.WriteString("\n  - ")
				sb.WriteString(msg.Message)
			}
			return nil, errors.New(sb.String())
		}
		return nil, errors.New("unable to create manifest")
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
