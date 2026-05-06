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
	Description     string              `json:"description,omitempty"`
	Attributes      map[string]string   `json:"attributes,omitempty"`
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
	response, err := getOciRepo(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCI repo %s: %w", id, err)
	}

	r := response.OciRepo
	repo := OciRepo{
		ID:           r.Id,
		Name:         r.Name,
		ArtifactType: r.ArtifactType,
		Description:  r.Description,
		Attributes:   anyMapToStringMap(r.Attributes),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	for _, rc := range r.ReleaseChannels.Items {
		repo.ReleaseChannels = append(repo.ReleaseChannels, OciReleaseChannel(rc))
	}
	for _, tag := range r.Tags.Items {
		repo.Tags = append(repo.Tags, OciRepoTag(tag))
	}

	return &repo, nil
}

// ListOciRepos returns all OCI repositories, optionally filtered and sorted.
// It automatically paginates through all pages.
func ListOciRepos(ctx context.Context, mdClient *client.Client, filter *OciReposFilter, sort *OciReposSort) ([]OciRepo, error) {
	var repos []OciRepo
	var cursor *Cursor

	for {
		response, err := listOciRepos(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, filter, sort, cursor)
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
				repo.ReleaseChannels = append(repo.ReleaseChannels, OciReleaseChannel(rc))
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

// CreateOciRepo creates a new OCI repository in the organization's catalog.
// `artifactType` selects the kind of artifact the repo will store (today only
// `BUNDLE`; resource-types and provisioners arrive later).
func CreateOciRepo(ctx context.Context, mdClient *client.Client, input CreateOciRepoInput) (*OciRepo, error) {
	response, err := createOciRepo(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, input)
	if err != nil {
		return nil, err
	}
	if !response.CreateOciRepo.Successful {
		messages := make([]string, 0, len(response.CreateOciRepo.Messages))
		for _, m := range response.CreateOciRepo.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to create OCI repo", messages)
	}
	r := response.CreateOciRepo.Result
	return &OciRepo{
		ID:           r.Id,
		Name:         r.Name,
		ArtifactType: r.ArtifactType,
		Attributes:   anyMapToStringMap(r.Attributes),
	}, nil
}

// UpdateOciRepo updates an OCI repository's mutable metadata (today: attributes).
func UpdateOciRepo(ctx context.Context, mdClient *client.Client, id string, input UpdateOciRepoInput) (*OciRepo, error) {
	response, err := updateOciRepo(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, id, input)
	if err != nil {
		return nil, err
	}
	if !response.UpdateOciRepo.Successful {
		messages := make([]string, 0, len(response.UpdateOciRepo.Messages))
		for _, m := range response.UpdateOciRepo.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to update OCI repo", messages)
	}
	r := response.UpdateOciRepo.Result
	return &OciRepo{
		ID:           r.Id,
		Name:         r.Name,
		ArtifactType: r.ArtifactType,
		Attributes:   anyMapToStringMap(r.Attributes),
	}, nil
}

// DeleteOciRepo removes an OCI repository. Refused by the server if the repo
// has any published versions.
func DeleteOciRepo(ctx context.Context, mdClient *client.Client, id string) (*OciRepo, error) {
	response, err := deleteOciRepo(ctx, mdClient.GQLv2, mdClient.Config.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if !response.DeleteOciRepo.Successful {
		messages := make([]string, 0, len(response.DeleteOciRepo.Messages))
		for _, m := range response.DeleteOciRepo.Messages {
			messages = append(messages, m.Message)
		}
		return nil, mutationError("unable to delete OCI repo", messages)
	}
	r := response.DeleteOciRepo.Result
	return &OciRepo{
		ID:           r.Id,
		Name:         r.Name,
		ArtifactType: r.ArtifactType,
	}, nil
}

// anyMapToStringMap coerces a `map[string]any` (the shape Map scalar fields
// land in) into the `map[string]string` shape attributes ride on. Non-string
// values are stringified via fmt.Sprintf.
func anyMapToStringMap(m map[string]any) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
