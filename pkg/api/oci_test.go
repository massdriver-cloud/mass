package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"

	"github.com/stretchr/testify/assert"
)

func TestGetOciRepo(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"ociRepo": map[string]any{
				"name": "aws-ecs-cluster",
				"tags": []map[string]any{
					{"tag": "1.0.0"},
					{"tag": "1.1.0"},
					{"tag": "1.1.1"},
				},
				"releaseChannels": []map[string]any{
					{
						"name": "~1",
						"tag":  "1.1.1",
					},
					{
						"name": "~1.0",
						"tag":  "1.0.0",
					},
					{
						"name": "~1.1",
						"tag":  "1.1.1",
					},
					{
						"name": "latest",
						"tag":  "1.1.1",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetOciRepo(t.Context(), &mdClient, "aws-ecs-cluster")
	if err != nil {
		t.Fatal(err)
	}

	want := &api.OciRepo{
		Name: "aws-ecs-cluster",
		Tags: []api.OciRepoTag{
			{Tag: "1.0.0"},
			{Tag: "1.1.0"},
			{Tag: "1.1.1"},
		},
		ReleaseChannels: []api.OciRepoReleaseChannel{
			{Name: "~1", Tag: "1.1.1"},
			{Name: "~1.0", Tag: "1.0.0"},
			{Name: "~1.1", Tag: "1.1.1"},
			{Name: "latest", Tag: "1.1.1"},
		},
	}

	assert.Equal(t, want, got)
}

func TestGetOciRepoTags(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"ociRepo": map[string]any{
				"tags": []map[string]any{
					{"tag": "1.0.0"},
					{"tag": "1.1.0"},
					{"tag": "1.1.1"},
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetOciRepoTags(t.Context(), &mdClient, "aws-ecs-cluster")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"1.0.0",
		"1.1.0",
		"1.1.1",
	}

	assert.ElementsMatch(t, got, want)
}
