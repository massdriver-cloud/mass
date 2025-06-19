package api_test

import (
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetProject(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"project": map[string]any{
				"id":   "uuid1",
				"slug": "sluggy",
				"defaultParams": map[string]any{
					"foo": "bar",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	project, err := api.GetProject(context.Background(), &mdClient, "sluggy")

	if err != nil {
		t.Fatal(err)
	}

	got := project.Slug

	want := "sluggy"

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}

func TestListProjects(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"projects": []map[string]any{
				{
					"id":   "uuid1",
					"name": "project1",
				},
				{
					"id":   "uuid2",
					"name": "project2",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	projects, err := api.ListProjects(context.Background(), &mdClient)

	if err != nil {
		t.Fatal(err)
	}

	got := len(projects)

	want := 2

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}
}
