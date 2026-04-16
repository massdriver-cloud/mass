package api_test

import (
	"testing"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetEnvironment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"environment": map[string]any{
				"id":          "env-uuid1",
				"name":        "Production",
				"description": "Production environment",
				"project": map[string]any{
					"id":   "proj-1",
					"name": "My Project",
				},
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	env, err := api.GetEnvironment(t.Context(), &mdClient, "env-uuid1")
	if err != nil {
		t.Fatal(err)
	}

	if env.ID != "env-uuid1" {
		t.Errorf("got %s, wanted env-uuid1", env.ID)
	}
	if env.Name != "Production" {
		t.Errorf("got %s, wanted Production", env.Name)
	}
	if env.Project == nil || env.Project.ID != "proj-1" {
		t.Errorf("expected project with ID proj-1")
	}
}

func TestListEnvironments(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"environments": map[string]any{
				"cursor": map[string]any{},
				"items": []map[string]any{
					{
						"id":   "env-1",
						"name": "staging",
					},
					{
						"id":   "env-2",
						"name": "production",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	envs, err := api.ListEnvironments(t.Context(), &mdClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(envs) != 2 {
		t.Errorf("got %d environments, wanted 2", len(envs))
	}
}

func TestCreateEnvironment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createEnvironment": map[string]any{
				"result": map[string]any{
					"id":          "env-new",
					"name":        "Staging",
					"description": "Staging environment",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	env, err := api.CreateEnvironment(t.Context(), &mdClient, "proj-1", api.CreateEnvironmentInput{
		Id:          "staging",
		Name:        "Staging",
		Description: "Staging environment",
	})
	if err != nil {
		t.Fatal(err)
	}

	if env.ID != "env-new" {
		t.Errorf("got %s, wanted env-new", env.ID)
	}
}

func TestSetEnvironmentDefault(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"setEnvironmentDefault": map[string]any{
				"result": map[string]any{
					"id": "envdef-1",
					"resource": map[string]any{
						"id":   "res-1",
						"name": "default-vpc",
						"resourceType": map[string]any{
							"id":   "aws-vpc",
							"name": "AWS VPC",
						},
					},
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	envDefault, err := api.SetEnvironmentDefault(t.Context(), &mdClient, "env-1", "res-1")
	if err != nil {
		t.Fatal(err)
	}

	if envDefault.ID != "envdef-1" {
		t.Errorf("got %s, wanted envdef-1", envDefault.ID)
	}
	if envDefault.Resource.ID != "res-1" {
		t.Errorf("got resource ID %s, wanted res-1", envDefault.Resource.ID)
	}
	if envDefault.Resource.Name != "default-vpc" {
		t.Errorf("got resource name %s, wanted default-vpc", envDefault.Resource.Name)
	}
	if envDefault.Resource.ResourceType == nil || envDefault.Resource.ResourceType.ID != "aws-vpc" {
		t.Errorf("expected resource type aws-vpc")
	}
}

func TestSetEnvironmentDefaultFailure(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"setEnvironmentDefault": map[string]any{
				"result":     nil,
				"successful": false,
				"messages": []map[string]any{
					{
						"code":    "conflict",
						"field":   "resourceId",
						"message": "a default of this resource type already exists",
					},
				},
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	_, err := api.SetEnvironmentDefault(t.Context(), &mdClient, "env-1", "res-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteEnvironment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"deleteEnvironment": map[string]any{
				"result": map[string]any{
					"id":   "env-1",
					"name": "Staging",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQLv1: gqlClient,
	}

	env, err := api.DeleteEnvironment(t.Context(), &mdClient, "env-1")
	if err != nil {
		t.Fatal(err)
	}

	if env.ID != "env-1" {
		t.Errorf("got %s, wanted env-1", env.ID)
	}
}
