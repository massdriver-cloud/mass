package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetOciRepo(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"ociRepo": map[string]any{
				"id":           "repo-uuid1",
				"name":         "aws-aurora-postgres",
				"artifactType": "application/vnd.massdriver.bundle.v1+json",
				"releaseChannels": map[string]any{
					"items": []map[string]any{
						{"name": "latest", "tag": "1.2.3"},
						{"name": "~1", "tag": "1.2.3"},
					},
				},
				"tags": map[string]any{
					"items": []map[string]any{
						{"tag": "1.2.3"},
						{"tag": "1.1.0"},
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	repo, err := api.GetOciRepo(t.Context(), &mdClient, "aws-aurora-postgres")
	if err != nil {
		t.Fatal(err)
	}

	if repo.ID != "repo-uuid1" {
		t.Errorf("got ID %s, wanted repo-uuid1", repo.ID)
	}
	if repo.Name != "aws-aurora-postgres" {
		t.Errorf("got name %s, wanted aws-aurora-postgres", repo.Name)
	}
	if repo.ArtifactType != "application/vnd.massdriver.bundle.v1+json" {
		t.Errorf("got artifact type %s", repo.ArtifactType)
	}
	if len(repo.ReleaseChannels) != 2 {
		t.Fatalf("got %d release channels, wanted 2", len(repo.ReleaseChannels))
	}
	if repo.ReleaseChannels[0].Name != "latest" || repo.ReleaseChannels[0].Tag != "1.2.3" {
		t.Errorf("got release channel %+v, wanted latest@1.2.3", repo.ReleaseChannels[0])
	}
	if len(repo.Tags) != 2 {
		t.Fatalf("got %d tags, wanted 2", len(repo.Tags))
	}
	if repo.Tags[0].Tag != "1.2.3" {
		t.Errorf("got tag %s, wanted 1.2.3", repo.Tags[0].Tag)
	}
}

func TestListOciRepos(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"ociRepos": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":           "repo-1",
						"name":         "aws-aurora-postgres",
						"artifactType": "application/vnd.massdriver.bundle.v1+json",
					},
					{
						"id":           "repo-2",
						"name":         "aws-s3",
						"artifactType": "application/vnd.massdriver.bundle.v1+json",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	repos, err := api.ListOciRepos(t.Context(), &mdClient, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(repos) != 2 {
		t.Errorf("got %d repos, wanted 2", len(repos))
	}
}

func TestListOciReposWithFilter(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"ociRepos": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":           "repo-1",
						"name":         "aws-aurora-postgres",
						"artifactType": "application/vnd.massdriver.bundle.v1+json",
					},
				},
			},
		},
	})
	mdClient := client.Client{GQLv1: gqlClient}

	filter := api.OciReposFilter{
		Name: &api.OciRepoNameFilter{StartsWith: "aws-"},
	}
	repos, err := api.ListOciRepos(t.Context(), &mdClient, &filter, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(repos) != 1 {
		t.Errorf("got %d repos, wanted 1", len(repos))
	}
}
