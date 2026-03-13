package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

// OciRepoReleaseChannel represents a release channel entry for an OCI repository.
type OciRepoReleaseChannel struct {
	Name string `json:"name" mapstructure:"name"`
	Tag  string `json:"tag" mapstructure:"tag"`
}

// OciRepoTag represents a single tag in an OCI repository.
type OciRepoTag struct {
	Tag string `json:"tag" mapstructure:"tag"`
}

// OciRepo represents an OCI container image repository with its tags and release channels.
type OciRepo struct {
	Name            string                  `json:"name" mapstructure:"name"`
	Tags            []OciRepoTag            `json:"tags" mapstructure:"tags"`
	ReleaseChannels []OciRepoReleaseChannel `json:"releaseChannels" mapstructure:"releaseChannels"`
}

// GetOciRepo retrieves an OCI repository by name from the Massdriver API.
func GetOciRepo(ctx context.Context, mdClient *client.Client, repo string) (*OciRepo, error) {
	response, err := getOciRepo(ctx, mdClient.GQL, mdClient.Config.OrganizationID, repo)
	if err != nil {
		return nil, err
	}

	return toOciRepo(response.OciRepo)
}

func toOciRepo(v any) (*OciRepo, error) {
	repo := OciRepo{}
	if err := mapstructure.Decode(v, &repo); err != nil {
		return nil, fmt.Errorf("failed to decode OCI repo: %w", err)
	}
	return &repo, nil
}

// GetOciRepoTags retrieves the list of tags for an OCI repository from the Massdriver API.
func GetOciRepoTags(ctx context.Context, mdClient *client.Client, repo string) ([]string, error) {
	response, err := getOciRepo(ctx, mdClient.GQL, mdClient.Config.OrganizationID, repo)
	if err != nil {
		return nil, err
	}

	return extractTags(response.OciRepo.Tags), nil
}

func extractTags(tags []getOciRepoOciRepoTagsOciTag) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Tag
	}
	return result
}
