package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
	"github.com/stretchr/testify/require"
)

func TestListRepos(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"repos": map[string]any{
				"cursor": map[string]any{
					"next":     "cursor123",
					"previous": "",
				},
				"items": []map[string]any{
					{
						"id":        "uuid1",
						"name":      "my-org/postgres",
						"createdAt": "2025-01-01T00:00:00Z",
						"releaseChannels": []map[string]any{
							{"name": "latest", "tag": "1.2.3"},
							{"name": "1.x", "tag": "1.2.3"},
						},
					},
					{
						"id":        "uuid2",
						"name":      "my-org/redis",
						"createdAt": "2025-01-02T00:00:00Z",
						"releaseChannels": []map[string]any{
							{"name": "latest", "tag": "2.0.0"},
						},
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
		Config: config.Config{
			OrganizationID: "org-123",
		},
	}

	page, err := api.ListRepos(t.Context(), &mdClient, api.ReposListOptions{})

	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, "cursor123", page.NextCursor)
	require.Empty(t, page.PrevCursor)

	require.Equal(t, "uuid1", page.Items[0].ID)
	require.Equal(t, "my-org/postgres", page.Items[0].Name)
	require.Len(t, page.Items[0].ReleaseChannels, 2)
	require.Equal(t, "latest", page.Items[0].ReleaseChannels[0].Name)
	require.Equal(t, "1.2.3", page.Items[0].ReleaseChannels[0].Tag)

	require.Equal(t, "uuid2", page.Items[1].ID)
	require.Equal(t, "my-org/redis", page.Items[1].Name)
}

func TestListReposWithOptions(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"repos": map[string]any{
				"cursor": map[string]any{
					"next":     "",
					"previous": "",
				},
				"items": []map[string]any{
					{
						"id":              "uuid1",
						"name":            "my-org/postgres",
						"createdAt":       "2025-01-01T00:00:00Z",
						"releaseChannels": []map[string]any{},
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
		Config: config.Config{
			OrganizationID: "org-123",
		},
	}

	opts := api.ReposListOptions{
		Search:    "postgres",
		SortField: "created_at",
		SortOrder: "desc",
		Limit:     10,
	}

	page, err := api.ListRepos(t.Context(), &mdClient, opts)

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "my-org/postgres", page.Items[0].Name)
}
