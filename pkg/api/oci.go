package api

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/mitchellh/mapstructure"
)

type OciRepoReleaseChannel struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type OciRepoTag struct {
	Tag string `json:"tag"`
}

type OciRepo struct {
	Name            string                  `json:"name"`
	Tags            []OciRepoTag            `json:"tags"`
	ReleaseChannels []OciRepoReleaseChannel `json:"releaseChannels"`
}

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
