package api

import (
	"context"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api/scalars"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Repo represents a bundle repository
type Repo struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	CreatedAt       time.Time        `json:"createdAt"`
	ReleaseChannels []ReleaseChannel `json:"releaseChannels"`
}

// ReleaseChannel represents a release channel
type ReleaseChannel struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// ReposPage represents a page of repos with pagination info
type ReposPage struct {
	Items      []Repo `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	PrevCursor string `json:"prevCursor,omitempty"`
}

// ReposListOptions contains options for listing repos
type ReposListOptions struct {
	Search    string
	SortField string // "name" or "created_at"
	SortOrder string // "asc" or "desc"
	Limit     int
}

// ListRepos lists bundle repositories with optional search, sort, and pagination
func ListRepos(ctx context.Context, mdClient *client.Client, opts ReposListOptions) (*ReposPage, error) {
	var sort *ReposSort
	if opts.SortField != "" || opts.SortOrder != "" {
		s := ReposSort{
			Field: ReposSortFieldName,
			Order: SortOrderAsc,
		}
		if opts.SortField == "created_at" {
			s.Field = ReposSortFieldCreatedAt
		}
		if opts.SortOrder == "desc" {
			s.Order = SortOrderDesc
		}
		sort = &s
	}

	var cursor *scalars.Cursor
	if opts.Limit > 0 {
		cursor = &scalars.Cursor{
			Limit: opts.Limit,
		}
	}

	var search *string
	if opts.Search != "" {
		search = &opts.Search
	}

	response, err := listRepos(ctx, mdClient.GQL, mdClient.Config.OrganizationID, sort, cursor, search)
	if err != nil {
		return nil, err
	}

	page := &ReposPage{
		Items:      make([]Repo, 0, len(response.Repos.Items)),
		NextCursor: response.Repos.Cursor.Next,
		PrevCursor: response.Repos.Cursor.Previous,
	}

	for _, item := range response.Repos.Items {
		repo := Repo{
			ID:              item.Id,
			Name:            item.Name,
			CreatedAt:       item.CreatedAt,
			ReleaseChannels: make([]ReleaseChannel, 0, len(item.ReleaseChannels)),
		}
		for _, rc := range item.ReleaseChannels {
			repo.ReleaseChannels = append(repo.ReleaseChannels, ReleaseChannel(rc))
		}
		page.Items = append(page.Items, repo)
	}

	return page, nil
}
