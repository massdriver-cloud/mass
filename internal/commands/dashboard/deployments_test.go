package dashboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func (f *fakeAPI) Deployments(_ context.Context, q dashboard.DeploymentQuery, _ string) ([]dashboard.Deployment, string, error) {
	f.scopeMu.Lock()
	defer f.scopeMu.Unlock()
	f.deployReads = append(f.deployReads, q)
	if f.deployErr != nil {
		return nil, "", f.deployErr
	}
	var out []dashboard.Deployment
	for _, d := range f.deploys {
		if slices.Contains(q.Instances, d.Instance.ID) && slices.Contains(q.Statuses, d.Status) {
			out = append(out, d)
		}
	}
	return out, "", nil
}

func (f *fakeAPI) PeekDeployment(_ context.Context, _, deploymentID string) (dashboard.DeploymentPeek, error) {
	return f.peeks[deploymentID], f.peekErr
}

func (f *fakeAPI) RejectDeployment(_ context.Context, deploymentID, reason string) error {
	f.rejected = append(f.rejected, deploymentID)
	f.reasons = append(f.reasons, reason)
	return f.actionErr
}

func deploy(id, status, inst, created string) dashboard.Deployment {
	return dashboard.Deployment{ID: id, Status: status, Action: "PROVISION", CreatedAt: created, Instance: ref(inst, "")}
}

// deskFixture adds deployments to the fixture: two awaiting approval, two
// in flight, two finished, and one outside the view.
func deskFixture() *fakeAPI {
	f := fixture()
	proposed := deploy("dep-1", "PROPOSED", "care-dev-queue", "2026-11-25T23:57:00Z")
	proposed.DeployedBy, proposed.Message = "ana@acme.dev", "Grow the queue\nfor the holidays"
	db := deploy("dep-db", "PROPOSED", "platform-dev-db", "2026-11-25T23:59:00Z")
	db.DeployedBy = "bo@acme.dev"
	old := deploy("dep-old", "COMPLETED", "care-dev-app", "2026-11-25T20:00:00Z")
	old.ElapsedTime, old.DeployedBy = 95, "cy@acme.dev"
	f.deploys = []dashboard.Deployment{
		proposed, db, old,
		deploy("dep-billing", "RUNNING", "billing-dev-app", "2026-11-25T23:58:00Z"),
		deploy("dep-net", "APPROVED", "platform-dev-network", "2026-11-25T23:50:00Z"),
		deploy("dep-fail", "FAILED", "care-dev-app", "2026-11-25T21:00:00Z"),
		deploy("dep-away", "COMPLETED", "faraway-app", "2026-11-25T23:00:00Z"),
	}
	return f
}

func deployIDs(ds []dashboard.Deployment) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.ID)
	}
	return out
}

func TestReadDeployments(t *testing.T) {
	many := make([]string, 0, 230)
	for i := range 230 {
		many = append(many, fmt.Sprintf("inst-%03d", i))
	}
	busy := fixture()
	for i := range 30 {
		busy.deploys = append(busy.deploys, deploy(fmt.Sprintf("done-%02d", i), "COMPLETED", "care-dev-app", fmt.Sprintf("2026-11-25T10:%02d:00Z", i)))
	}
	tests := []struct {
		name         string
		api          *fakeAPI
		instances    []string
		wantAwaiting []string
		wantInFlight []string
		wantRecent   []string
		wantReads    int
		wantErr      bool
	}{
		{
			name: "grouped and newest first", api: deskFixture(),
			instances:    []string{"care-dev-queue", "platform-dev-db", "care-dev-app", "billing-dev-app", "platform-dev-network"},
			wantAwaiting: []string{"dep-db", "dep-1"}, wantInFlight: []string{"dep-billing", "dep-net"}, wantRecent: []string{"dep-fail", "dep-old"},
			wantReads: 2,
		},
		{name: "nothing in view", api: deskFixture(), wantReads: 0},
		{name: "asks for instances in batches", api: fixture(), instances: many, wantReads: 6},
		{
			name: "keeps the newest of the history", api: busy, instances: []string{"care-dev-app"}, wantReads: 2,
			wantRecent: []string{
				"done-29", "done-28", "done-27", "done-26", "done-25", "done-24", "done-23", "done-22", "done-21", "done-20",
				"done-19", "done-18", "done-17", "done-16", "done-15", "done-14", "done-13", "done-12", "done-11", "done-10",
			},
		},
		{name: "a failed read", api: &fakeAPI{deployErr: errors.New("boom")}, instances: []string{"x"}, wantReads: 1, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			board, err := dashboard.ReadDeployments(context.Background(), tc.api, tc.instances)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
			if !slices.Equal(deployIDs(board.Awaiting), tc.wantAwaiting) || !slices.Equal(deployIDs(board.InFlight), tc.wantInFlight) ||
				!slices.Equal(deployIDs(board.Recent), tc.wantRecent) {
				t.Errorf("awaiting %v, in flight %v, recent %v", deployIDs(board.Awaiting), deployIDs(board.InFlight), deployIDs(board.Recent))
			}
			if len(tc.api.deployReads) != tc.wantReads {
				t.Errorf("%d reads, want %d", len(tc.api.deployReads), tc.wantReads)
			}
			for _, q := range tc.api.deployReads {
				if len(q.Instances) > 100 {
					t.Errorf("one read asked for %d instances", len(q.Instances))
				}
			}
		})
	}
}

func onDesk(t *testing.T, api *fakeAPI, cfg dashboard.Config) dashboard.Model {
	t.Helper()
	m, cmd := pressCmd(loaded(t, api, cfg, 200, 48), "5")
	return drive(t, m, cmd, 4)
}

func TestDeploymentsTab(t *testing.T) {
	api := deskFixture()
	m := loaded(t, api, config(&opener{}), 200, 48)
	if len(api.deployReads) != 0 {
		t.Fatalf("read deployments %d times before the tab opened", len(api.deployReads))
	}
	m, cmd := pressCmd(m, "5")
	m = drive(t, m, cmd, 4)
	if m.ActiveTab() != dashboard.TabDeployments || m.SelectedDeployment() != "dep-db" {
		t.Fatalf("tab %d, selected %q", m.ActiveTab(), m.SelectedDeployment())
	}
	out := view(m)
	order := []string{
		"5 Deployments", "AWAITING APPROVAL (2)", "platform-dev-db", "bo@acme.dev", "care-dev-queue", "Care / Development",
		"ana@acme.dev", "3m ago", "Grow the queue", "IN FLIGHT (2)", "billing-dev-app", "running 2:00", "platform-dev-network",
		"approved", "RECENT", "failed", "completed 1:35", "4h ago", "cy@acme.dev", "c config", "x reject",
	}
	at := 0
	for _, want := range order {
		i := strings.Index(out[at:], want)
		if i < 0 {
			t.Fatalf("view is missing %q after %q:\n%s", want, out[:at], out)
		}
		at += i + len(want)
	}
	if strings.Contains(out, "faraway-app") || strings.Contains(out, "for the holidays") {
		t.Errorf("view shows a deployment out of view or a message's second line:\n%s", out)
	}

	reads := len(api.deployReads)
	m, cmd = pressCmd(m, "r")
	m = drive(t, m, cmd, 20)
	if len(api.deployReads) <= reads {
		t.Errorf("a refresh with the tab open did not read deployments again")
	}
	m = press(m, "1")
	reads = len(api.deployReads)
	m, cmd = pressCmd(m, "r")
	drive(t, m, cmd, 20)
	if len(api.deployReads) != reads {
		t.Errorf("a refresh with the tab closed read deployments")
	}
}

func TestDeploymentsReadFails(t *testing.T) {
	api := deskFixture()
	api.deployErr = errors.New("gateway timeout")
	m := onDesk(t, api, config(&opener{}))
	if out := view(m); !strings.Contains(out, "could not read deployments: list deployments: gateway timeout · r retries") {
		t.Fatalf("view is missing the retry hint:\n%s", out)
	}
	api.deployErr = nil
	m, cmd := pressCmd(m, "r")
	if out := view(drive(t, m, cmd, 20)); !strings.Contains(out, "AWAITING APPROVAL (2)") || strings.Contains(out, "r retries") {
		t.Errorf("r should read the deployments again:\n%s", out)
	}
}

func TestDeploymentsDuringReplay(t *testing.T) {
	cfg := config(&opener{})
	cfg.Replay = time.Hour
	if out := view(onDesk(t, deskFixture(), cfg)); !strings.Contains(out, "this tab lists live deployments") {
		t.Errorf("view is missing the replay note:\n%s", out)
	}
	if out := view(onDesk(t, deskFixture(), config(&opener{}))); strings.Contains(out, "this tab lists live deployments") {
		t.Errorf("the replay note shows with no replay:\n%s", out)
	}
}

func TestDeploymentSelectionPersists(t *testing.T) {
	api := deskFixture()
	m := press(onDesk(t, api, config(&opener{})), "down")
	if m.SelectedDeployment() != "dep-1" {
		t.Fatalf("selected %q", m.SelectedDeployment())
	}
	newer := deploy("dep-new", "PROPOSED", "care-dev-app", "2026-11-25T23:59:30Z")
	api.deploys = append(api.deploys, newer)
	m, cmd := pressCmd(m, "r")
	m = drive(t, m, cmd, 20)
	if m.SelectedDeployment() != "dep-1" {
		t.Errorf("selected %q after a refresh added a row above it, want dep-1", m.SelectedDeployment())
	}
	api.deploys = slices.DeleteFunc(api.deploys, func(d dashboard.Deployment) bool { return d.ID == "dep-1" })
	m, cmd = pressCmd(m, "r")
	m = drive(t, m, cmd, 20)
	if m.SelectedDeployment() != "dep-billing" {
		t.Errorf("selected %q after the selected row left, want the row now in its place", m.SelectedDeployment())
	}
	if m = press(m, "up", "up", "up"); m.SelectedDeployment() != "dep-old" {
		t.Errorf("selected %q after wrapping up past the top", m.SelectedDeployment())
	}
}

func TestDeploymentKeys(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		readOnly     bool
		setup        func(*fakeAPI)
		wantTab      dashboard.Tab
		wantSelected string
		wantApproved []string
		wantRejected []string
		wantReasons  []string
		wantView     []string
		notWant      []string
	}{
		{name: "enter shows it on the map", keys: []string{"down", "enter"}, wantTab: dashboard.TabConstellation, wantSelected: "care-dev"},
		{
			name: "approve", keys: []string{"down", "y", "y"}, wantTab: dashboard.TabDeployments, wantApproved: []string{"dep-1"},
			wantView: []string{"✓ approved care-dev-queue"},
		},
		{
			name: "approve asks first", keys: []string{"down", "y"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"APPROVE DEPLOYMENT", "care-dev-queue", "“Grow the queue”", "Approve the proposed provision?", "c peek the config"},
		},
		{
			name: "peek from the approval and come back", keys: []string{"down", "y", "c", "esc"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"APPROVE DEPLOYMENT", "y/enter approve"}, notWant: []string{"CONFIG"},
		},
		{
			name: "approve after a peek", keys: []string{"down", "y", "c", "esc", "enter"}, wantTab: dashboard.TabDeployments,
			wantApproved: []string{"dep-1"},
		},
		{
			name: "reject with a reason", keys: []string{"down", "x", "t", "o", "o", " ", "b", "i", "g", "x", "backspace", "enter"},
			wantTab: dashboard.TabDeployments, wantRejected: []string{"dep-1"}, wantReasons: []string{"too big"},
			wantView: []string{"✓ rejected care-dev-queue"},
		},
		{
			name: "reject without a reason", keys: []string{"x", "enter"}, wantTab: dashboard.TabDeployments,
			wantRejected: []string{"dep-db"}, wantReasons: []string{""}, wantView: []string{"✓ rejected platform-dev-db"},
		},
		{
			name: "reject asks for a reason", keys: []string{"x", "n", "o", "p", "e"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"REJECT DEPLOYMENT", "platform-dev-db", "Reason (optional): nope", "enter reject", "esc cancel"},
		},
		{name: "reject cancelled", keys: []string{"x", "n", "esc"}, wantTab: dashboard.TabDeployments, wantView: []string{"reject cancelled"}},
		{
			name: "reason not saved", keys: []string{"x", "w", "h", "y", "enter"}, wantTab: dashboard.TabDeployments,
			setup: func(f *fakeAPI) {
				f.actionErr = fmt.Errorf("%w: set stash entry: forbidden", dashboard.ErrReasonNotSaved)
			},
			wantRejected: []string{"dep-db"}, wantReasons: []string{"why"},
			wantView: []string{"✓ rejected platform-dev-db, but the reason was not saved: set stash entry: forbidden"},
		},
		{
			name: "reject refused", keys: []string{"x", "enter"}, wantTab: dashboard.TabDeployments,
			setup:        func(f *fakeAPI) { f.actionErr = fmt.Errorf("reject: %w", gql.ErrForbidden) },
			wantRejected: []string{"dep-db"}, wantReasons: []string{""},
			wantView: []string{"✕ you do not have permission to reject platform-dev-db"},
		},
		{name: "read-only approve", keys: []string{"y", "y"}, readOnly: true, wantTab: dashboard.TabDeployments, wantView: []string{"read-only; restart it without --read-only to approve"}},
		{name: "read-only reject", keys: []string{"x"}, readOnly: true, wantTab: dashboard.TabDeployments, wantView: []string{"read-only; restart it without --read-only to reject"}},
		{
			name: "only a proposal can be approved", keys: []string{"down", "down", "y"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"billing-dev-app's deployment is running; only a proposed one can be approved"},
		},
		{
			name: "only a proposal can be rejected", keys: []string{"up", "x"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"care-dev-app's deployment is completed; only a proposed one can be rejected"},
		},
		{
			name: "follow logs full width", keys: []string{"down", "down", "f"}, wantTab: dashboard.TabDeployments,
			setup: func(f *fakeAPI) {
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {{Status: "RUNNING", Batches: []dashboard.LogBatch{batch(1, "terraform apply")}}}}
			},
			wantView: []string{"LOGS app", "provision · ⟳ running", "terraform apply"}, notWant: []string{"AWAITING APPROVAL"},
		},
		{
			name: "moving on closes the logs", keys: []string{"down", "down", "f", "down"}, wantTab: dashboard.TabDeployments,
			setup: func(f *fakeAPI) {
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {{Status: "RUNNING"}}}
			},
			wantView: []string{"AWAITING APPROVAL"}, notWant: []string{"LOGS app"},
		},
		{
			name: "peek", keys: []string{"down", "c"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"CONFIG care-dev-queue", "Care / Development", "bundle 1.0.0 → 1.1.0", "MESSAGE", "Grow the queue", "for the holidays"},
		},
		{
			name: "peek fails", keys: []string{"c"}, wantTab: dashboard.TabDeployments,
			setup:    func(f *fakeAPI) { f.peekErr = errors.New("not found\x1b[2J") },
			wantView: []string{"could not read the config: not found"}, notWant: []string{"\x1b[2J"},
		},
		{name: "esc closes the peek", keys: []string{"c", "esc"}, wantTab: dashboard.TabDeployments, wantView: []string{"AWAITING APPROVAL"}, notWant: []string{"CONFIG"}},
		{
			name: "o opens the instance", keys: []string{"down", "o"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"↗ opened https://app/inst/care-dev-queue"},
		},
		{
			name: "help", keys: []string{"?"}, wantTab: dashboard.TabDeployments,
			wantView: []string{"DEPLOYMENTS", "with secrets masked", "stash as review/rejection"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := deskFixture()
			api.peeks = map[string]dashboard.DeploymentPeek{"dep-1": peekFixture()}
			if tc.setup != nil {
				tc.setup(api)
			}
			cfg := logConfig()
			cfg.ReadOnly = tc.readOnly
			m := onDesk(t, api, cfg)
			m, cmd := keys(m, tc.keys...)
			m = drive(t, m, cmd, 8)
			if m.ActiveTab() != tc.wantTab {
				t.Errorf("tab = %d, want %d", m.ActiveTab(), tc.wantTab)
			}
			if tc.wantSelected != "" && m.Selected() != tc.wantSelected {
				t.Errorf("selected %q, want %q", m.Selected(), tc.wantSelected)
			}
			if !slices.Equal(api.approved, tc.wantApproved) || !slices.Equal(api.rejected, tc.wantRejected) || !slices.Equal(api.reasons, tc.wantReasons) {
				t.Errorf("approved %v, rejected %v with %q; want %v, %v with %q", api.approved, api.rejected, api.reasons, tc.wantApproved, tc.wantRejected, tc.wantReasons)
			}
			out := view(m)
			for _, want := range tc.wantView {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			for _, bad := range tc.notWant {
				if strings.Contains(out, bad) {
					t.Errorf("view has %q:\n%s", bad, out)
				}
			}
		})
	}
}

func peekFixture() dashboard.DeploymentPeek {
	changed := func(path, from, to string) types.ParamComparison {
		return types.ParamComparison{Path: path, Source: types.ParamValue{Present: true, Value: from}, Target: types.ParamValue{Present: true, Value: to}}
	}
	return dashboard.DeploymentPeek{
		Deployment: types.Deployment{
			ID: "dep-1", Action: "PROVISION", Status: "PROPOSED", Version: "1.1.0", DeployedBy: "ana@acme.dev",
			Message: "Grow the queue\nfor the holidays", CreatedAt: fixedNow().Add(-3 * 60e9),
		},
		Baseline: &dashboard.Deployment{ID: "dep-0", CreatedAt: "2026-11-24T00:00:00Z"},
		Version:  types.VersionComparison{Source: "1.0.0", Target: "1.1.0"},
		Params: []types.ParamComparison{
			changed(".size", "small", "large"),
			changed(".db.password", "hunter2", "correct-horse"),
			changed(".notes", "plain", "-----BEGIN RSA PRIVATE KEY-----"),
			{Path: ".replicas", Target: types.ParamValue{Present: true, Value: "3"}},
			{Path: ".legacy", Source: types.ParamValue{Present: true, Value: "on"}},
			{Path: ".api_token", Target: types.ParamValue{Present: true, Value: "tok_live"}},
			{Path: ".database.dsn", Target: types.ParamValue{Present: true, Value: "host=db user=app"}},
			{Path: ".queue.connection_string", Target: types.ParamValue{Present: true, Value: "Endpoint=sb://q"}},
			{Path: ".webhook", Target: types.ParamValue{Present: true, Value: "https://bot:s3cr3t@hooks.example.com/x"}},
			{Path: ".homepage", Target: types.ParamValue{Present: true, Value: "https://example.com/x"}},
			{Path: ".region", Source: types.ParamValue{Present: true, Value: "us-east-1"}, Target: types.ParamValue{Present: true, Value: "us-east-1"}, Equal: true},
		},
	}
}

func TestPeekRendering(t *testing.T) {
	tests := []struct {
		name    string
		peek    dashboard.DeploymentPeek
		want    []string
		notWant []string
	}{
		{
			name: "diff against the baseline",
			peek: peekFixture(),
			want: []string{
				"provision · proposed · by ana@acme.dev · 3m ago", "against the last completed provision, 2d ago",
				"~ .db.password •••••• → ••••••", "- .legacy on", "~ .notes plain → ••••••", "+ .replicas 3",
				"~ .size small → large", "+ .api_token ••••••", "+ .database.dsn ••••••", "+ .queue.connection_string ••••••",
				"+ .webhook ••••••", "+ .homepage https://example.com/x", "1 param unchanged",
			},
			notWant: []string{"hunter2", "correct-horse", "BEGIN RSA", "tok_live", ".region", "user=app", "sb://q", "s3cr3t"},
		},
		{
			name: "no baseline",
			peek: dashboard.DeploymentPeek{
				Deployment: types.Deployment{ID: "dep-1", Action: "PROVISION", Status: "PROPOSED", Version: "1.1.0"},
				Version:    types.VersionComparison{Target: "1.1.0"},
				Params:     []types.ParamComparison{{Path: ".size", Target: types.ParamValue{Present: true, Value: "large"}}},
			},
			want: []string{"bundle 1.1.0", "no message", "no completed provision before it, so every param is new", "+ .size large"},
		},
		{
			name: "nothing changed",
			peek: dashboard.DeploymentPeek{Baseline: &dashboard.Deployment{ID: "dep-0"}, Version: types.VersionComparison{Source: "1.0.0", Target: "1.0.0", Equal: true}},
			want: []string{"bundle 1.0.0", "no param changes"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := deskFixture()
			api.peeks = map[string]dashboard.DeploymentPeek{"dep-1": tc.peek}
			m, cmd := keys(onDesk(t, api, config(&opener{})), "down", "c")
			out := view(drive(t, m, cmd, 4))
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			for _, bad := range tc.notWant {
				if strings.Contains(out, bad) {
					t.Errorf("view shows %q:\n%s", bad, out)
				}
			}
		})
	}
}

func TestPeekScrolls(t *testing.T) {
	api := deskFixture()
	peek := peekFixture()
	for i := range 60 {
		peek.Params = append(peek.Params, types.ParamComparison{Path: fmt.Sprintf(".zz%02d", i), Target: types.ParamValue{Present: true, Value: "v"}})
	}
	api.peeks = map[string]dashboard.DeploymentPeek{"dep-1": peek}
	m, cmd := keys(onDesk(t, api, config(&opener{})), "down", "c")
	m = drive(t, m, cmd, 4)
	if out := view(m); !strings.Contains(out, "CONFIG") || !strings.Contains(out, "more") || strings.Contains(out, ".zz59") {
		t.Fatalf("a long config should open at the top:\n%s", out)
	}
	m = press(m, "G")
	if out := view(m); !strings.Contains(out, ".zz59") || strings.Contains(out, "CONFIG") {
		t.Errorf("end should scroll to the bottom:\n%s", out)
	}
	m = press(m, "g", "j")
	if out := view(m); strings.Contains(out, "CONFIG") {
		t.Errorf("j should scroll down a line:\n%s", out)
	}
}

func TestAPIDeployments(t *testing.T) {
	client := &recordingClient{data: map[string]string{
		"dashboardDeployments": `{"deployments":{"cursor":{"next":"c2"},"items":[
			{"id":"d2","status":"COMPLETED","action":"PROVISION","createdAt":"2026-11-25T00:00:00Z","instance":{"id":"i1"}},
			{"id":"d1","status":"COMPLETED","action":"PROVISION","createdAt":"2026-11-24T00:00:00Z","instance":{"id":"i1"}}]}}`,
	}}
	d := &fakeDeployer{
		got:  &types.Deployment{ID: "d2", Version: "1.1.0", Params: map[string]any{"size": "large"}, CreatedAt: fixedNow()},
		diff: &types.DeploymentComparison{Version: types.VersionComparison{Source: "1.0.0", Target: "1.1.0"}},
	}
	api := dashboard.NewAPIWithClient(client, dashboard.Services{Audit: &fakeAudit{}, Deployer: d}, "acme")
	ctx := context.Background()

	got, next, err := api.Deployments(ctx, dashboard.DeploymentQuery{Instances: []string{"i1"}, Statuses: []string{"RUNNING"}}, "c1")
	if err != nil || next != "c2" || len(got) != 2 || got[0].Instance.ID != "i1" {
		t.Fatalf("Deployments = %v, %q, %v", got, next, err)
	}
	vars, _ := client.reqs[0].Variables.(map[string]any)
	if got := fmt.Sprint(vars["filter"]); got != "map[instanceId:map[in:[i1]] status:map[in:[RUNNING]]]" {
		t.Errorf("filter = %s", got)
	}

	peek, err := api.PeekDeployment(ctx, "i1", "d2")
	if err != nil || peek.Baseline == nil || peek.Baseline.ID != "d1" || peek.Version.Source != "1.0.0" {
		t.Fatalf("PeekDeployment = %+v, %v; want it compared with the provision before it", peek, err)
	}
	if !slices.Equal(d.compared, [][2]string{{"d1", "d2"}}) {
		t.Errorf("compared %v", d.compared)
	}

	d.got = &types.Deployment{ID: "d0", Version: "0.1.0", Params: map[string]any{"db": map[string]any{"port": 5432}, "tags": []any{"a"}, "on": true}, CreatedAt: fixedNow().AddDate(-1, 0, 0)}
	peek, err = api.PeekDeployment(ctx, "i1", "d0")
	if err != nil || peek.Baseline != nil {
		t.Fatalf("PeekDeployment = %+v, %v; want no baseline older than the deployment", peek, err)
	}
	var leaves []string
	for _, p := range peek.Params {
		leaves = append(leaves, p.Path+"="+p.Target.Value)
	}
	if got := strings.Join(leaves, " "); got != ".db.port=5432 .on=true .tags[0]=a" {
		t.Errorf("params %s", got)
	}
}

// pagedClient serves pages of completed provisions, the next page named
// by the cursor, and keeps asking for more after the last one given.
type pagedClient struct {
	pages   []string
	filters []string
}

func (c *pagedClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	vars, _ := req.Variables.(map[string]any)
	c.filters = append(c.filters, fmt.Sprint(vars["filter"]))
	cursor, _ := vars["cursor"].(map[string]any)
	n := 0
	if next, ok := cursor["next"].(string); ok {
		n, _ = strconv.Atoi(next)
	}
	items := "[]"
	if n < len(c.pages) {
		items = c.pages[n]
	}
	raw, _ := resp.Data.(*json.RawMessage)
	*raw = json.RawMessage(fmt.Sprintf(`{"deployments":{"cursor":{"next":"%d"},"items":%s}}`, n+1, items))
	return nil
}

func provisions(created ...string) string {
	items := make([]string, 0, len(created))
	for _, at := range created {
		items = append(items, fmt.Sprintf(`{"id":"p-%s","status":"COMPLETED","action":"PROVISION","createdAt":"%s"}`, at[:10], at))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func TestPeekBaseline(t *testing.T) {
	newer := provisions("2026-11-25T03:00:00Z", "2026-11-25T02:00:00Z", "2026-11-25T01:00:00Z")
	tests := []struct {
		name      string
		pages     []string
		wantBase  string
		wantReads int
	}{
		{name: "on the first page", pages: []string{provisions("2026-11-25T03:00:00Z", "2026-11-20T00:00:00Z")}, wantBase: "p-2026-11-20", wantReads: 1},
		{name: "past page one", pages: []string{newer, newer, provisions("2026-11-19T00:00:00Z")}, wantBase: "p-2026-11-19", wantReads: 3},
		{name: "none within reach", pages: []string{newer, newer, newer, newer, newer, provisions("2026-11-01T00:00:00Z")}, wantReads: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &pagedClient{pages: tc.pages}
			d := &fakeDeployer{got: &types.Deployment{ID: "target", CreatedAt: fixedNow().AddDate(0, 0, -5)}, diff: &types.DeploymentComparison{}}
			api := dashboard.NewAPIWithClient(client, dashboard.Services{Audit: &fakeAudit{}, Deployer: d}, "acme")
			peek, err := api.PeekDeployment(context.Background(), "i1", "target")
			if err != nil {
				t.Fatal(err)
			}
			base := ""
			if peek.Baseline != nil {
				base = peek.Baseline.ID
			}
			if base != tc.wantBase || len(client.filters) != tc.wantReads {
				t.Errorf("baseline %q after %d reads, want %q after %d", base, len(client.filters), tc.wantBase, tc.wantReads)
			}
			if want := "map[action:map[in:[PROVISION]] instanceId:map[in:[i1]] status:map[in:[COMPLETED]]]"; client.filters[0] != want {
				t.Errorf("filter = %s", client.filters[0])
			}
		})
	}
}

func TestAPIRejectDeployment(t *testing.T) {
	tests := []struct {
		name        string
		reason      string
		stashErr    error
		rejectErr   error
		wantStashed int
		wantErr     error
	}{
		{name: "with a reason", reason: "too big", wantStashed: 1},
		{name: "without a reason"},
		{name: "reason not saved", reason: "too big", stashErr: errors.New("forbidden"), wantStashed: 1, wantErr: dashboard.ErrReasonNotSaved},
		{name: "a refused reject keeps no reason", reason: "too big", rejectErr: gql.ErrForbidden, wantErr: gql.ErrForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &fakeDeployer{stashErr: tc.stashErr, rejectErr: tc.rejectErr}
			api := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: &fakeAudit{}, Deployer: d}, "acme")
			err := api.RejectDeployment(context.Background(), "dep-1", tc.reason)
			if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if !slices.Equal(d.rejected, []string{"dep-1"}) {
				t.Errorf("rejected %v", d.rejected)
			}
			if len(d.stashed) != tc.wantStashed {
				t.Fatalf("stashed %d times", len(d.stashed))
			}
			if tc.wantStashed > 0 {
				want := stashSet{scope: stash.Scope{Kind: stash.ScopeDeployment, ID: "dep-1"}, key: "review/rejection", in: stash.SetInput{Value: tc.reason, Format: stash.FormatText}}
				if d.stashed[0] != want {
					t.Errorf("stashed %+v, want %+v", d.stashed[0], want)
				}
			}
		})
	}

	bare := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: &fakeAudit{}}, "acme")
	if err := bare.RejectDeployment(context.Background(), "x", ""); !errors.Is(err, dashboard.ErrNoDeployer) {
		t.Errorf("RejectDeployment without a deployer = %v", err)
	}
	if _, err := bare.PeekDeployment(context.Background(), "i", "x"); !errors.Is(err, dashboard.ErrNoDeployer) {
		t.Errorf("PeekDeployment without a deployer = %v", err)
	}
}
