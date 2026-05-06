package instance_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestRunDeployReusesLastConfig(t *testing.T) {
	var createDeploymentVars map[string]any

	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"instance": map[string]any{
						"id":     "inst-1",
						"name":   "cache",
						"status": "PROVISIONED",
						"params": map[string]any{"size": "small"},
					},
				},
			}
		},
		func(req *http.Request) any {
			createDeploymentVars = gqlmock.ParseInputVariables(req)
			return map[string]any{
				"data": map[string]any{
					"createDeployment": map[string]any{
						"successful": true,
						"result": map[string]any{
							"id":     "dep-1",
							"status": "PENDING",
						},
					},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"deployment": map[string]any{
						"id":     "dep-1",
						"status": "COMPLETED",
					},
				},
			}
		},
	}

	mdClient := client.Client{GQLv2: gqlmock.NewClientWithFuncResponseArray(responses)}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	dep, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", instance.DeployOptions{Message: "redeploy"})
	if err != nil {
		t.Fatal(err)
	}

	if dep.Status != "COMPLETED" {
		t.Errorf("got %s, wanted COMPLETED", dep.Status)
	}

	input, ok := createDeploymentVars["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input map, got %T", createDeploymentVars["input"])
	}
	if input["message"] != "redeploy" {
		t.Errorf("expected message 'redeploy', got %v", input["message"])
	}
	if input["action"] != "PROVISION" {
		t.Errorf("expected action 'PROVISION', got %v", input["action"])
	}

	gotParams := map[string]any{}
	gqlmock.MustUnmarshalJSON([]byte(input["params"].(string)), &gotParams)
	wantParams := map[string]any{"size": "small"}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("got params %v, wanted %v", gotParams, wantParams)
	}
}

func TestRunDeployWithParamsReplacesConfig(t *testing.T) {
	var createDeploymentVars map[string]any

	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"instance": map[string]any{
						"id":     "inst-1",
						"params": map[string]any{"size": "small"},
					},
				},
			}
		},
		func(req *http.Request) any {
			createDeploymentVars = gqlmock.ParseInputVariables(req)
			return map[string]any{
				"data": map[string]any{
					"createDeployment": map[string]any{
						"successful": true,
						"result":     map[string]any{"id": "dep-1", "status": "PENDING"},
					},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"deployment": map[string]any{"id": "dep-1", "status": "COMPLETED"},
				},
			}
		},
	}

	mdClient := client.Client{GQLv2: gqlmock.NewClientWithFuncResponseArray(responses)}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	t.Setenv("MEMORY_AMT", "6")
	opts := instance.DeployOptions{
		Params: map[string]any{"size": "${MEMORY_AMT}GB"},
	}

	_, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", opts)
	if err != nil {
		t.Fatal(err)
	}

	input := createDeploymentVars["input"].(map[string]any)
	gotParams := map[string]any{}
	gqlmock.MustUnmarshalJSON([]byte(input["params"].(string)), &gotParams)
	wantParams := map[string]any{"size": "6GB"}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("got params %v, wanted %v", gotParams, wantParams)
	}
}

func TestRunDeployWithPatchQueriesUpdatesLastConfig(t *testing.T) {
	var createDeploymentVars map[string]any

	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"instance": map[string]any{
						"id":     "inst-1",
						"params": map[string]any{"cidr": "10.0.0.0/16", "name": "keep"},
					},
				},
			}
		},
		func(req *http.Request) any {
			createDeploymentVars = gqlmock.ParseInputVariables(req)
			return map[string]any{
				"data": map[string]any{
					"createDeployment": map[string]any{
						"successful": true,
						"result":     map[string]any{"id": "dep-1", "status": "PENDING"},
					},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"deployment": map[string]any{"id": "dep-1", "status": "COMPLETED"},
				},
			}
		},
	}

	mdClient := client.Client{GQLv2: gqlmock.NewClientWithFuncResponseArray(responses)}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	opts := instance.DeployOptions{
		PatchQueries: []string{`.cidr = "10.0.0.0/20"`},
	}

	_, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", opts)
	if err != nil {
		t.Fatal(err)
	}

	input := createDeploymentVars["input"].(map[string]any)
	gotParams := map[string]any{}
	gqlmock.MustUnmarshalJSON([]byte(input["params"].(string)), &gotParams)
	wantParams := map[string]any{"cidr": "10.0.0.0/20", "name": "keep"}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("got params %v, wanted %v", gotParams, wantParams)
	}
}

func TestRunDeployWithDecommissionAction(t *testing.T) {
	var createDeploymentVars map[string]any

	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"instance": map[string]any{
						"id":     "inst-1",
						"params": map[string]any{"size": "small"},
					},
				},
			}
		},
		func(req *http.Request) any {
			createDeploymentVars = gqlmock.ParseInputVariables(req)
			return map[string]any{
				"data": map[string]any{
					"createDeployment": map[string]any{
						"successful": true,
						"result":     map[string]any{"id": "dep-1", "status": "PENDING"},
					},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"deployment": map[string]any{"id": "dep-1", "status": "COMPLETED"},
				},
			}
		},
	}

	mdClient := client.Client{GQLv2: gqlmock.NewClientWithFuncResponseArray(responses)}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	_, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", instance.DeployOptions{
		Action: api.DeploymentActionDecommission,
	})
	if err != nil {
		t.Fatal(err)
	}

	input := createDeploymentVars["input"].(map[string]any)
	if input["action"] != "DECOMMISSION" {
		t.Errorf("expected action 'DECOMMISSION', got %v", input["action"])
	}

	gotParams := map[string]any{}
	gqlmock.MustUnmarshalJSON([]byte(input["params"].(string)), &gotParams)
	wantParams := map[string]any{"size": "small"}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("got params %v, wanted %v", gotParams, wantParams)
	}
}

func TestRunDeployFailsWhenDeploymentFails(t *testing.T) {
	responses := []gqlmock.ResponseFunc{
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"instance": map[string]any{"id": "inst-1", "params": map[string]any{}},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"createDeployment": map[string]any{
						"successful": true,
						"result":     map[string]any{"id": "dep-1", "status": "PENDING"},
					},
				},
			}
		},
		func(req *http.Request) any {
			return map[string]any{
				"data": map[string]any{
					"deployment": map[string]any{"id": "dep-1", "status": "FAILED"},
				},
			}
		},
	}

	mdClient := client.Client{GQLv2: gqlmock.NewClientWithFuncResponseArray(responses)}
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	if _, err := instance.RunDeploy(t.Context(), &mdClient, "ecomm-prod-cache", instance.DeployOptions{}); err == nil {
		t.Fatal("expected error, got nil")
	}
}
