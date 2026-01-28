package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func floatPtr(v float64) *float64 { return &v }

func TestGetEnvironment(t *testing.T) {
	want := api.Environment{
		ID:          "env-uuid1",
		Name:        "Test Environment",
		Slug:        "env",
		Description: "This is a test environment",
		Cost: &api.Cost{
			Daily: api.Summary{
				Average: api.CostSample{
					Amount: floatPtr(10.0),
				},
			},
			Monthly: api.Summary{
				Average: api.CostSample{
					Amount: floatPtr(300.0),
				},
			},
		},
		Packages: []api.Package{
			{
				ID:     "pkg-uuid1",
				Params: map[string]any{"param1": "value1"},
				Artifacts: []api.Artifact{
					{
						ID:    "artifact-uuid1",
						Name:  "artifact1",
						Field: "field1",
					},
				},
				RemoteReferences: []api.RemoteReference{
					{
						Artifact: api.Artifact{
							ID:    "remote-artifact-uuid1",
							Name:  "remote-artifact1",
							Field: "remote-field1",
						},
					},
				},
				Bundle: &api.Bundle{
					ID:   "bundle-uuid1",
					Name: "test-bundle",
				},
				Status: string(api.PackageStatusProvisioned),
				Manifest: &api.Manifest{
					ID:          "manifest-uuid1",
					Name:        "Test Manifest",
					Slug:        "manifest",
					Suffix:      "0000",
					Description: "This is a test manifest",
				},
			},
		},
		Project: &api.Project{
			ID:   "proj-uuid1",
			Slug: "proj",
		},
	}
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"environment": map[string]any{
				"id":          "env-uuid1",
				"name":        "Test Environment",
				"slug":        "env",
				"description": "This is a test environment",
				"cost": map[string]any{
					"daily": map[string]any{
						"average": map[string]any{
							"amount": 10.0,
						},
					},
					"monthly": map[string]any{
						"average": map[string]any{
							"amount": 300.0,
						},
					},
				},
				"packages": []map[string]any{
					{
						"id":     "pkg-uuid1",
						"params": map[string]any{"param1": "value1"},
						"artifacts": []map[string]any{
							{
								"id":    "artifact-uuid1",
								"name":  "artifact1",
								"field": "field1",
							},
						},
						"remoteReferences": []map[string]any{
							{
								"artifact": map[string]any{
									"id":    "remote-artifact-uuid1",
									"name":  "remote-artifact1",
									"field": "remote-field1",
								},
							},
						},
						"bundle": map[string]any{
							"id":   "bundle-uuid1",
							"name": "test-bundle",
						},
						"status": "PROVISIONED",
						"manifest": map[string]any{
							"id":          "manifest-uuid1",
							"name":        "Test Manifest",
							"slug":        "manifest",
							"suffix":      "0000",
							"description": "This is a test manifest",
						},
					},
				},
				"project": map[string]any{
					"id":   "proj-uuid1",
					"slug": "proj",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetEnvironment(t.Context(), &mdClient, "proj-env")

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, &want) {
		t.Errorf("got %+v, wanted %+v", got, &want)
	}
}

func TestGetEnvironmentsByPackage(t *testing.T) {
	packageId := "pkg-uuid1"

	want := []api.Environment{
		{
			ID:          "env-uuid1",
			Name:        "Test Environment 1",
			Slug:        "env1",
			Description: "First test environment",
			Cost: &api.Cost{
				Daily: api.Summary{
					Average: api.CostSample{
						Amount: floatPtr(5.0),
					},
				},
				Monthly: api.Summary{
					Average: api.CostSample{
						Amount: floatPtr(150.0),
					},
				},
			},
			Project: &api.Project{
				ID:   "proj-uuid1",
				Slug: "proj1",
			},
		},
		{
			ID:          "env-uuid2",
			Name:        "Test Environment 2",
			Slug:        "env2",
			Description: "Second test environment",
			Cost: &api.Cost{
				Daily: api.Summary{
					Average: api.CostSample{
						Amount: floatPtr(8.0),
					},
				},
				Monthly: api.Summary{
					Average: api.CostSample{
						Amount: floatPtr(240.0),
					},
				},
			},
			Project: &api.Project{
				ID:   "proj-uuid2",
				Slug: "proj2",
			},
		},
	}

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"project": map[string]any{
				"environments": []map[string]any{
					{
						"id":          "env-uuid1",
						"name":        "Test Environment 1",
						"slug":        "env1",
						"description": "First test environment",
						"cost": map[string]any{
							"daily": map[string]any{
								"average": map[string]any{
									"amount": 5.0,
								},
							},
							"monthly": map[string]any{
								"average": map[string]any{
									"amount": 150.0,
								},
							},
						},
						"project": map[string]any{
							"id":   "proj-uuid1",
							"slug": "proj1",
						},
					},
					{
						"id":          "env-uuid2",
						"name":        "Test Environment 2",
						"slug":        "env2",
						"description": "Second test environment",
						"cost": map[string]any{
							"daily": map[string]any{
								"average": map[string]any{
									"amount": 8.0,
								},
							},
							"monthly": map[string]any{
								"average": map[string]any{
									"amount": 240.0,
								},
							},
						},
						"project": map[string]any{
							"id":   "proj-uuid2",
							"slug": "proj2",
						},
					},
				},
			},
		},
	})

	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetEnvironmentsByProject(t.Context(), &mdClient, packageId)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, wanted %+v", got, want)
	}
}
