package instance_test

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// fakeDeployAPI is a hand-rolled stub for instance.DeployAPI. Each test sets
// the fields it cares about; the captured Create input is what most assertions
// inspect.
type fakeDeployAPI struct {
	instance       *types.Instance
	getInstanceErr error

	deployment      *types.Deployment
	createErr       error
	gotCreateInput  deployments.CreateInput
	gotCreateInstID string

	finalDeployment  *types.Deployment
	getDeploymentErr error
}

func (f *fakeDeployAPI) GetInstance(_ context.Context, id string) (*types.Instance, error) {
	if f.getInstanceErr != nil {
		return nil, f.getInstanceErr
	}
	// stamp the requested ID onto a copy so callers don't have to pre-set it.
	inst := *f.instance
	if inst.ID == "" {
		inst.ID = id
	}
	return &inst, nil
}

func (f *fakeDeployAPI) CreateDeployment(_ context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error) {
	f.gotCreateInstID = instanceID
	f.gotCreateInput = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.deployment, nil
}

func (f *fakeDeployAPI) GetDeployment(_ context.Context, _ string) (*types.Deployment, error) {
	if f.getDeploymentErr != nil {
		return nil, f.getDeploymentErr
	}
	return f.finalDeployment, nil
}

func (f *fakeDeployAPI) TailLogs(_ context.Context, _ string, _ io.Writer) error {
	// Not used by these tests (none set LogWriter).
	return nil
}

// newDeployFake spins up a fake wired for the happy-path shape: one instance
// to fetch, one deployment to return on Create, and one final deployment status
// to return on the post-create Get poll.
func newDeployFake(instanceParams map[string]any, finalStatus string) *fakeDeployAPI {
	return &fakeDeployAPI{
		instance: &types.Instance{
			ID:     "inst-1",
			Name:   "cache",
			Status: "PROVISIONED",
			Params: instanceParams,
		},
		deployment:      &types.Deployment{ID: "dep-1", Status: "PENDING"},
		finalDeployment: &types.Deployment{ID: "dep-1", Status: finalStatus},
	}
}

func TestRunDeployReusesLastConfig(t *testing.T) {
	api := newDeployFake(map[string]any{"size": "small"}, "COMPLETED")
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	dep, err := instance.RunDeploy(t.Context(), api, "ecomm-prod-cache", instance.DeployOptions{Message: "redeploy"})
	if err != nil {
		t.Fatal(err)
	}
	if dep.Status != "COMPLETED" {
		t.Errorf("got %s, wanted COMPLETED", dep.Status)
	}

	if api.gotCreateInput.Message != "redeploy" {
		t.Errorf("expected message 'redeploy', got %q", api.gotCreateInput.Message)
	}
	if api.gotCreateInput.Action != deployments.ActionProvision {
		t.Errorf("expected action PROVISION, got %q", api.gotCreateInput.Action)
	}

	wantParams := map[string]any{"size": "small"}
	if !reflect.DeepEqual(api.gotCreateInput.Params, wantParams) {
		t.Errorf("got params %v, wanted %v", api.gotCreateInput.Params, wantParams)
	}
}

func TestRunDeployWithParamsReplacesConfig(t *testing.T) {
	api := newDeployFake(map[string]any{"size": "small"}, "COMPLETED")
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	t.Setenv("MEMORY_AMT", "6")
	opts := instance.DeployOptions{
		Params: map[string]any{"size": "${MEMORY_AMT}GB"},
	}

	if _, err := instance.RunDeploy(t.Context(), api, "ecomm-prod-cache", opts); err != nil {
		t.Fatal(err)
	}

	wantParams := map[string]any{"size": "6GB"}
	if !reflect.DeepEqual(api.gotCreateInput.Params, wantParams) {
		t.Errorf("got params %v, wanted %v", api.gotCreateInput.Params, wantParams)
	}
}

func TestRunDeployWithPatchQueriesUpdatesLastConfig(t *testing.T) {
	api := newDeployFake(map[string]any{"cidr": "10.0.0.0/16", "name": "keep"}, "COMPLETED")
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	opts := instance.DeployOptions{
		PatchQueries: []string{`.cidr = "10.0.0.0/20"`},
	}

	if _, err := instance.RunDeploy(t.Context(), api, "ecomm-prod-cache", opts); err != nil {
		t.Fatal(err)
	}

	wantParams := map[string]any{"cidr": "10.0.0.0/20", "name": "keep"}
	if !reflect.DeepEqual(api.gotCreateInput.Params, wantParams) {
		t.Errorf("got params %v, wanted %v", api.gotCreateInput.Params, wantParams)
	}
}

func TestRunDeployWithDecommissionAction(t *testing.T) {
	api := newDeployFake(map[string]any{"size": "small"}, "COMPLETED")
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	_, err := instance.RunDeploy(t.Context(), api, "ecomm-prod-cache", instance.DeployOptions{
		Action: deployments.ActionDecommission,
	})
	if err != nil {
		t.Fatal(err)
	}

	if api.gotCreateInput.Action != deployments.ActionDecommission {
		t.Errorf("expected action DECOMMISSION, got %q", api.gotCreateInput.Action)
	}

	wantParams := map[string]any{"size": "small"}
	if !reflect.DeepEqual(api.gotCreateInput.Params, wantParams) {
		t.Errorf("got params %v, wanted %v", api.gotCreateInput.Params, wantParams)
	}
}

func TestRunDeployFailsWhenDeploymentFails(t *testing.T) {
	api := newDeployFake(map[string]any{}, "FAILED")
	instance.DeploymentStatusSleep = 0 //nolint:reassign // intentionally overriding sleep duration in tests

	if _, err := instance.RunDeploy(t.Context(), api, "ecomm-prod-cache", instance.DeployOptions{}); err == nil {
		t.Fatal("expected error, got nil")
	}
}
