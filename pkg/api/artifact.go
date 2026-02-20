package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type RemoteReference struct {
	Artifact Artifact `json:"artifact"`
}

type Artifact struct {
	ID                 string                        `json:"id"`
	Name               string                        `json:"name"`
	Type               string                        `json:"type"`
	Field              string                        `json:"field,omitempty"`
	Payload            map[string]any                `json:"payload,omitempty"`
	Formats            []string                      `json:"formats,omitempty"`
	CreatedAt          time.Time                     `json:"createdAt,omitempty"`
	UpdatedAt          time.Time                     `json:"updatedAt,omitempty"`
	ArtifactDefinition *ArtifactDefinitionWithSchema `json:"artifactDefinition,omitempty"`
	Package            *ArtifactPackage              `json:"package,omitempty"`
	Origin             string                        `json:"origin,omitempty"`
}

// ArtifactPackage is a minimal representation of a Package containing only ID and Slug.
// We use a separate struct instead of the full Package struct because Package has required
// non-omitempty fields (Status, Params) that we don't have when getting artifact details.
type ArtifactPackage struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

func CreateArtifact(ctx context.Context, mdClient *client.Client, artifactName string, artifactType string, artifactPayload map[string]any) (*Artifact, error) {
	response, err := createArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactName, artifactType, artifactPayload)
	if err != nil {
		return nil, err
	}
	if !response.CreateArtifact.Successful {
		return nil, fmt.Errorf("unable to create artifact: %s", response.CreateArtifact.GetMessages()[0].Message)
	}
	return response.CreateArtifact.toArtifact(), err
}

func (payload *createArtifactCreateArtifactArtifactPayload) toArtifact() *Artifact {
	return &Artifact{
		Name: payload.Result.Name,
		ID:   payload.Result.Id,
	}
}

func DownloadArtifact(ctx context.Context, mdClient *client.Client, artifactID string, format string) (string, error) {
	response, err := downloadArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID, format)
	if err != nil {
		return "", fmt.Errorf("failed to download artifact %s: %w", artifactID, err)
	}

	return response.DownloadArtifact.RenderedArtifact, nil
}

func GetArtifact(ctx context.Context, mdClient *client.Client, artifactID string) (*Artifact, error) {
	response, err := getArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact %s: %w", artifactID, err)
	}
	return response.toArtifact(), nil
}

func (response *getArtifactResponse) toArtifact() *Artifact {
	artifact := &Artifact{
		ID:        response.Artifact.Id,
		Name:      response.Artifact.Name,
		Type:      response.Artifact.Type,
		Field:     response.Artifact.Field,
		Payload:   response.Artifact.Payload,
		Formats:   response.Artifact.Formats,
		CreatedAt: response.Artifact.CreatedAt,
		UpdatedAt: response.Artifact.UpdatedAt,
		Origin:    string(response.Artifact.Origin),
	}

	// ArtifactDefinition is always present (non-nullable in GraphQL schema)
	artifact.ArtifactDefinition = &ArtifactDefinitionWithSchema{
		ID:    response.Artifact.ArtifactDefinition.Id,
		Name:  response.Artifact.ArtifactDefinition.Name,
		Label: response.Artifact.ArtifactDefinition.Label,
	}

	// Package may be null, check if it has a non-empty ID
	if response.Artifact.Package.Id != "" {
		artifact.Package = &ArtifactPackage{
			ID:   response.Artifact.Package.Id,
			Slug: response.Artifact.Package.Slug,
		}
	}

	return artifact
}

func UpdateArtifact(ctx context.Context, mdClient *client.Client, artifactID string, artifactName string, artifactPayload map[string]any) (*Artifact, error) {
	response, err := updateArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID, artifactName, artifactPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to update artifact %s: %w", artifactID, err)
	}
	if !response.UpdateArtifact.Successful {
		messages := response.UpdateArtifact.GetMessages()
		if len(messages) > 0 {
			errMsg := "unable to update artifact:"
			for _, msg := range messages {
				errMsg += "\n  - " + msg.Message
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("unable to update artifact")
	}
	if response.UpdateArtifact.Result.Id == "" {
		return nil, fmt.Errorf("update artifact returned no result")
	}
	return &Artifact{
		ID:   response.UpdateArtifact.Result.Id,
		Name: response.UpdateArtifact.Result.Name,
	}, nil
}

func DeleteArtifact(ctx context.Context, mdClient *client.Client, artifactID string) (*Artifact, error) {
	response, err := deleteArtifact(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete artifact %s: %w", artifactID, err)
	}
	if !response.DeleteArtifact.Successful {
		messages := response.DeleteArtifact.GetMessages()
		if len(messages) > 0 {
			errMsg := "unable to delete artifact:"
			for _, msg := range messages {
				errMsg += "\n  - " + msg.Message
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("unable to delete artifact")
	}
	// Check if result is empty (genqlient generates value types, not pointers)
	if response.DeleteArtifact.Result.Id == "" {
		return nil, fmt.Errorf("delete artifact returned no result")
	}
	return &Artifact{
		ID:   response.DeleteArtifact.Result.Id,
		Name: response.DeleteArtifact.Result.Name,
	}, nil
}
