package api

import (
	"context"
	"fmt"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// OciRepo is an OCI repository in your organization's catalog.
type OciRepo struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	ArtifactType    string              `json:"artifactType"`
	CreatedAt       time.Time           `json:"createdAt,omitzero"`
	UpdatedAt       time.Time           `json:"updatedAt,omitzero"`
	ReleaseChannels []OciReleaseChannel `json:"releaseChannels,omitempty"`
	Tags            []OciRepoTag        `json:"tags,omitempty"`
}

// OciReleaseChannel is a release channel that auto-resolves to the latest matching version.
type OciReleaseChannel struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// OciRepoTag is a published version tag in an OCI repository.
type OciRepoTag struct {
	Tag       string    `json:"tag"`
	CreatedAt time.Time `json:"createdAt,omitzero"`
}

// GetOciRepo retrieves an OCI repository by name.
func GetOciRepo(ctx context.Context, mdClient *client.Client, id string) (*OciRepo, error) {
	response, err := getOciRepo(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCI repo %s: %w", id, err)
	}

	r := response.OciRepo
	repo := OciRepo{
		ID:           r.Id,
		Name:         r.Name,
		ArtifactType: r.ArtifactType,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	for _, rc := range r.ReleaseChannels.Items {
		repo.ReleaseChannels = append(repo.ReleaseChannels, OciReleaseChannel{Name: rc.Name, Tag: rc.Tag})
	}
	for _, tag := range r.Tags.Items {
		repo.Tags = append(repo.Tags, OciRepoTag{Tag: tag.Tag, CreatedAt: tag.CreatedAt})
	}

	return &repo, nil
}

// ListOciRepos returns all OCI repositories, optionally filtered and sorted.
// It automatically paginates through all pages.
func ListOciRepos(ctx context.Context, mdClient *client.Client, filter *OciReposFilter, sort *OciReposSort) ([]OciRepo, error) {
	var repos []OciRepo
	var cursor *Cursor

	for {
		response, err := listOciRepos(ctx, mdClient.GQLv1, mdClient.Config.OrganizationID, filter, sort, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list OCI repos: %w", err)
		}

		for _, r := range response.OciRepos.Items {
			repo := OciRepo{
				ID:           r.Id,
				Name:         r.Name,
				ArtifactType: r.ArtifactType,
				CreatedAt:    r.CreatedAt,
				UpdatedAt:    r.UpdatedAt,
			}
			for _, rc := range r.ReleaseChannels.Items {
				repo.ReleaseChannels = append(repo.ReleaseChannels, OciReleaseChannel{Name: rc.Name, Tag: rc.Tag})
			}
			repos = append(repos, repo)
		}

		next := response.OciRepos.Cursor.Next
		if next == "" {
			break
		}
		cursor = &Cursor{Next: next}
	}

	return repos, nil
}
