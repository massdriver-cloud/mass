package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestActions(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		readOnly     bool
		err          error
		wantApproved []string
		wantDeployed []string
		wantView     []string
		wantTab      dashboard.Tab
	}{
		{name: "approve with y", keys: []string{"n", "n", "y", "y"}, wantApproved: []string{"dep-1"}, wantView: []string{"✓ approved care-dev-queue"}},
		{name: "approve with enter", keys: []string{"n", "n", "y", "enter"}, wantApproved: []string{"dep-1"}, wantView: []string{"✓ approved care-dev-queue"}},
		{name: "pick backwards", keys: []string{"N", "y", "y"}, wantApproved: []string{"dep-1"}},
		{name: "approve cancelled", keys: []string{"n", "n", "y", "x"}, wantView: []string{"approve cancelled"}},
		{name: "esc cancels", keys: []string{"n", "n", "y", "esc"}, wantView: []string{"approve cancelled"}},
		{name: "deploy with y", keys: []string{"n", "d", "y"}, wantDeployed: []string{"care-dev-app"}, wantView: []string{"⟳ deploying care-dev-app"}},
		{name: "deploy cancelled", keys: []string{"n", "d", "n"}, wantView: []string{"deploy cancelled"}},
		{name: "box swallows other keys", keys: []string{"n", "d", "2"}, wantView: []string{"deploy cancelled"}},
		{name: "approve needs a proposal", keys: []string{"n", "y", "y"}, wantView: []string{"care-dev-app has no deployment awaiting approval"}},
		{name: "nothing picked", keys: []string{"d", "y"}, wantView: []string{"pick an instance with n first"}},
		{name: "read-only approve", keys: []string{"n", "n", "y", "y"}, readOnly: true, wantView: []string{"the dashboard is read-only"}},
		{name: "read-only deploy", keys: []string{"n", "d", "y"}, readOnly: true, wantView: []string{"the dashboard is read-only"}},
		{
			name: "forbidden", keys: []string{"n", "d", "y"}, err: fmt.Errorf("create deployment: %w", gql.ErrForbidden),
			wantDeployed: []string{"care-dev-app"}, wantView: []string{"✕ you do not have permission to deploy care-dev-app"},
		},
		{
			name: "failure", keys: []string{"n", "n", "y", "y"}, err: errors.New("deployment is not PROPOSED\x1b[2J"),
			wantApproved: []string{"dep-1"}, wantView: []string{"✕ could not approve care-dev-queue: deployment is not PROPOSED"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := fixture()
			api.actionErr = tc.err
			cfg := config(&opener{})
			cfg.ReadOnly = tc.readOnly
			m := loaded(t, api, cfg, 200, 48)
			reads := len(api.envReads)
			m, cmd := keys(m, tc.keys...)
			m = drive(t, m, cmd, 6)
			if !slices.Equal(api.approved, tc.wantApproved) || !slices.Equal(api.deployed, tc.wantDeployed) {
				t.Errorf("approved %v, deployed %v; want %v, %v", api.approved, api.deployed, tc.wantApproved, tc.wantDeployed)
			}
			acted := len(tc.wantApproved)+len(tc.wantDeployed) > 0
			if reread := len(api.envReads) > reads; reread != acted {
				t.Errorf("re-read = %v, want %v", reread, acted)
			}
			if m.ActiveTab() != tc.wantTab {
				t.Errorf("tab = %d, want %d", m.ActiveTab(), tc.wantTab)
			}
			out := view(m)
			for _, want := range tc.wantView {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "any other key cancels") {
				t.Errorf("the confirmation is still open:\n%s", out)
			}
		})
	}
}

func TestActionConfirmation(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want []string
	}{
		{
			name: "approve", keys: []string{"n", "n", "y"},
			want: []string{"APPROVE DEPLOYMENT", "care-dev-queue", "Care / Development", "Approve the proposed provision?", "nothing depends on it downstream", "y/enter approve"},
		},
		{
			name: "deploy", keys: []string{"n", "d"},
			want: []string{"DEPLOY", "care-dev-app", "Care / Development", "Deploy a new deployment of its current configuration?", "1 environment and 1 instance depend on it downstream"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := fixture()
			m, _ := keys(loaded(t, api, config(&opener{}), 200, 48), tc.keys...)
			out := view(m)
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			if len(api.approved)+len(api.deployed) > 0 {
				t.Errorf("called before confirming: approved %v, deployed %v", api.approved, api.deployed)
			}
		})
	}
}

type fakeDeployer struct {
	inst     *types.Instance
	created  []deployments.CreateInput
	createID string
	approved []string
	err      error
}

func (f *fakeDeployer) GetInstance(_ context.Context, id string) (*types.Instance, error) {
	if f.err != nil {
		return nil, f.err
	}
	inst := *f.inst
	inst.ID = id
	return &inst, nil
}

func (f *fakeDeployer) CreateDeployment(_ context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error) {
	f.createID = instanceID
	f.created = append(f.created, in)
	return &types.Deployment{}, nil
}

func (f *fakeDeployer) ApproveDeployment(_ context.Context, id string) (*types.Deployment, error) {
	f.approved = append(f.approved, id)
	return &types.Deployment{}, f.err
}

func TestAPIActions(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]any
		wantParams map[string]any
	}{
		{name: "current params", params: map[string]any{"size": "large"}, wantParams: map[string]any{"size": "large"}},
		{name: "no params", wantParams: map[string]any{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &fakeDeployer{inst: &types.Instance{Params: tc.params}}
			api := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: &fakeAudit{}, Deployer: d}, "acme")
			if err := api.Deploy(context.Background(), "care-dev-app"); err != nil {
				t.Fatal(err)
			}
			if len(d.created) != 1 || d.createID != "care-dev-app" || d.created[0].Action != deployments.ActionProvision ||
				fmt.Sprint(d.created[0].Params) != fmt.Sprint(tc.wantParams) {
				t.Errorf("created %+v for %q", d.created, d.createID)
			}
		})
	}

	d := &fakeDeployer{}
	api := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: &fakeAudit{}, Deployer: d}, "acme")
	if err := api.ApproveDeployment(context.Background(), "dep-1"); err != nil || !slices.Equal(d.approved, []string{"dep-1"}) {
		t.Errorf("approved %v, %v", d.approved, err)
	}
	d.err = gql.ErrForbidden
	if err := api.Deploy(context.Background(), "care-dev-app"); !errors.Is(err, gql.ErrForbidden) || len(d.created) > 0 {
		t.Errorf("Deploy = %v after a refused read, created %v", err, d.created)
	}

	bare := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: &fakeAudit{}}, "acme")
	if err := bare.Deploy(context.Background(), "x"); !errors.Is(err, dashboard.ErrNoDeployer) {
		t.Errorf("Deploy without a deployer = %v", err)
	}
	if err := bare.ApproveDeployment(context.Background(), "x"); !errors.Is(err, dashboard.ErrNoDeployer) {
		t.Errorf("ApproveDeployment without a deployer = %v", err)
	}
}

func TestActionsFollowTheSelection(t *testing.T) {
	tests := map[string]struct {
		keys []string
		want string
	}{
		"a pick is cleared on leaving": {keys: []string{"n", "tab", "d"}, want: "pick an instance with n first"},
		"a deploy in flight":           {keys: []string{"BILLING", "n", "d"}, want: "billing-dev-app is already deploying"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api := fixture()
			m := loaded(t, api, config(&opener{}), 200, 48)
			for _, k := range tc.keys {
				if k == "BILLING" {
					m, _ = tabTo(t, m, "billing-dev")
					continue
				}
				m = press(m, k)
			}
			if out := view(m); !strings.Contains(out, tc.want) || len(api.deployed) != 0 {
				t.Errorf("deployed %v; view is missing %q:\n%s", api.deployed, tc.want, out)
			}
		})
	}
}
