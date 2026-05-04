package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetProject(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"project": map[string]any{
				"id":          "proj-uuid1",
				"name":        "My Project",
				"description": "A test project",
			},
		},
	})
	mdClient := client.Client{
		GQLv2: gqlClient,
	}

	project, err := api.GetProject(t.Context(), &mdClient, "proj-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if project.ID != "proj-uuid1" {
		t.Errorf("got %s, wanted proj-uuid1", project.ID)
	}
	if project.Name != "My Project" {
		t.Errorf("got %s, wanted My Project", project.Name)
	}
}

func TestListProjects(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"projects": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
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
		},
	})
	mdClient := client.Client{
		GQLv2: gqlClient,
	}

	projects, err := api.ListProjects(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 2 {
		t.Errorf("got %d projects, wanted 2", len(projects))
	}
}

func TestCreateProject(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createProject": map[string]any{
				"result": map[string]any{
					"id":          "new-proj",
					"name":        "New Project",
					"description": "A new project",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQLv2: gqlClient,
	}

	project, err := api.CreateProject(t.Context(), &mdClient, api.CreateProjectInput{
		Id:          "new-proj",
		Name:        "New Project",
		Description: "A new project",
	})
	if err != nil {
		t.Fatal(err)
	}

	if project.ID != "new-proj" {
		t.Errorf("got %s, wanted new-proj", project.ID)
	}
	if project.Name != "New Project" {
		t.Errorf("got %s, wanted New Project", project.Name)
	}
}

func TestCreateProjectFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createProject": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{
						"code":    "required",
						"field":   "name",
						"message": "name is required",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQLv2: gqlClient,
	}

	_, err := api.CreateProject(t.Context(), &mdClient, api.CreateProjectInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteProject(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deleteProject": map[string]any{
				"result": map[string]any{
					"id":   "proj-1",
					"name": "Deleted Project",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQLv2: gqlClient,
	}

	project, err := api.DeleteProject(t.Context(), &mdClient, "proj-1")
	if err != nil {
		t.Fatal(err)
	}

	if project.ID != "proj-1" {
		t.Errorf("got %s, wanted proj-1", project.ID)
	}
}
