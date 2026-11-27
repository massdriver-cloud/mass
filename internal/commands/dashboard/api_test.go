package dashboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/auditlogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

type fakeAPI struct {
	envPages   [][]dashboard.Environment
	runtimes   []dashboard.Runtime
	apps       []dashboard.Application
	err        error
	cursors    []string
	eventReads [][]dashboard.Event
	eventCalls int
	history    []dashboard.Event
	historyErr error
	since      time.Time
}

func (f *fakeAPI) History(_ context.Context, since time.Time) ([]dashboard.Event, error) {
	f.since = since
	return f.history, f.historyErr
}

func (f *fakeAPI) Events(context.Context, time.Time) ([]dashboard.Event, error) {
	f.eventCalls++
	if f.eventCalls > len(f.eventReads) {
		return nil, nil
	}
	return f.eventReads[f.eventCalls-1], nil
}

func (f *fakeAPI) Environments(_ context.Context, cursor string) ([]dashboard.Environment, string, error) {
	f.cursors = append(f.cursors, cursor)
	if f.err != nil {
		return nil, "", f.err
	}
	idx := 0
	if cursor != "" {
		idx = int(cursor[0] - '0')
	}
	next := ""
	if idx+1 < len(f.envPages) {
		next = string(rune('0' + idx + 1))
	}
	return f.envPages[idx], next, nil
}

func (f *fakeAPI) Runtimes(context.Context, string) ([]dashboard.Runtime, string, error) {
	return f.runtimes, "", nil
}

func (f *fakeAPI) Applications(context.Context, string) ([]dashboard.Application, string, error) {
	return f.apps, "", nil
}

func usd(amount float64) dashboard.Cost {
	return dashboard.Cost{LastMonth: dashboard.CostSample{Amount: &amount, Currency: "USD"}}
}

func ref(id, name string) *dashboard.Ref { return &dashboard.Ref{ID: id, Name: name} }

func dependsOn(instanceID, envID string) dashboard.Dependency {
	return dashboard.Dependency{Field: "upstream", Resource: dashboard.Resource{
		ID: instanceID + ".out", Origin: "PROVISIONED",
		Instance: &dashboard.ResourceInstance{ID: instanceID, Environment: ref(envID, "")},
	}}
}

func appBundle(name, version, runtimeVersion string) *dashboard.Bundle {
	return &dashboard.Bundle{Name: name, Version: version, Runtime: &dashboard.Bundle{Name: "ecs", Version: runtimeVersion}}
}

// fixture is a small organization: platform feeds care, care feeds billing,
// a database in platform is firing, and a sandbox was forked from a template.
func fixture() *fakeAPI {
	firing := dashboard.AlarmsPage{Items: []dashboard.Alarm{
		{DisplayName: "CPU high", CurrentState: &dashboard.AlarmState{Status: "ALARM", Message: "CPU over 80%", OccurredAt: "2026-11-25T00:00:00Z"}},
		{DisplayName: "Disk", CurrentState: &dashboard.AlarmState{Status: "OK"}},
	}}
	platform := dashboard.Environment{ID: "platform-dev", Name: "Development", Project: ref("platform", "Platform"), Instances: []dashboard.Instance{
		{ID: "platform-dev-network", Status: "PROVISIONED"},
		{ID: "platform-dev-db", Status: "PROVISIONED", Alarms: firing, Cost: usd(1250), Dependencies: []dashboard.Dependency{dependsOn("platform-dev-network", "platform-dev")}},
	}}
	care := dashboard.Environment{ID: "care-dev", Name: "Development", Project: ref("care", "Care"), Instances: []dashboard.Instance{
		{ID: "care-dev-app", Status: "PROVISIONED", Bundle: appBundle("care-app", "0.1.0", "1.0.0"),
			DeployedVersion: "0.1.0", AvailableUpgrade: "0.2.0",
			Dependencies: []dashboard.Dependency{dependsOn("platform-dev-db", "platform-dev"), dependsOn("secret-vpc", "secret-prod")}},
		{ID: "care-dev-queue", Status: "INITIALIZED", AvailableUpgrade: "1.0.0",
			LatestDeployment: &dashboard.Deployment{Status: "PROPOSED", Action: "PROVISION"}},
	}}
	billing := dashboard.Environment{ID: "billing-dev", Name: "Development", Project: ref("billing", "Billing"), Instances: []dashboard.Instance{
		{ID: "billing-dev-app", Status: "FAILED", Bundle: appBundle("billing-app", "2.0.0", "1.0.0"),
			LatestDeployment: &dashboard.Deployment{Status: "RUNNING", Action: "PROVISION", CreatedAt: "2026-11-25T23:58:00Z"},
			LatestProvision:  &dashboard.Provision{Bundle: appBundle("billing-app", "1.9.0", "0.9.0")},
			Dependencies:     []dashboard.Dependency{dependsOn("care-dev-app", "care-dev")}},
	}}
	template := dashboard.Environment{ID: "sandbox-template", Name: "Template", Project: ref("sandbox", "Sandboxes")}
	demo := dashboard.Environment{ID: "sandbox-demo", Name: "demo", Project: ref("sandbox", "Sandboxes"), Parent: ref("sandbox-template", "")}
	return &fakeAPI{
		envPages: [][]dashboard.Environment{{platform, care, billing}, {template, demo}},
		runtimes: []dashboard.Runtime{{ID: "ecs", Latest: dashboard.Bundle{Name: "ecs", Version: "1.0.0"}}},
		apps: []dashboard.Application{
			{ID: "care-app", Latest: *appBundle("care-app", "0.1.0", "1.0.0")},
			{ID: "billing-app", Latest: *appBundle("billing-app", "2.0.0", "0.9.0")},
			{ID: "lambda-app", Latest: dashboard.Bundle{Name: "lambda-app", Version: "1.0.0", Runtime: &dashboard.Bundle{Name: "lambda", Version: "1.0.0"}}},
		},
	}
}

func fixtureSnapshot(t *testing.T) *dashboard.Snapshot {
	t.Helper()
	snap, err := dashboard.Fetch(context.Background(), fixture())
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestFetch(t *testing.T) {
	tests := []struct {
		name     string
		api      *fakeAPI
		wantEnvs int
		wantErr  bool
	}{
		{name: "reads every page", api: fixture(), wantEnvs: 5},
		{name: "reports a failed page", api: &fakeAPI{err: errors.New("boom")}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap, err := dashboard.Fetch(context.Background(), tc.api)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := len(snap.Environments); got != tc.wantEnvs {
				t.Errorf("environments = %d, want %d", got, tc.wantEnvs)
			}
			if want := []string{"", "1"}; len(tc.api.cursors) != 2 || tc.api.cursors[1] != want[1] {
				t.Errorf("cursors = %v, want %v", tc.api.cursors, want)
			}
		})
	}
}

type recordingClient struct {
	reqs []*graphql.Request
	data map[string]string
}

func (r *recordingClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	r.reqs = append(r.reqs, req)
	raw, ok := resp.Data.(*json.RawMessage)
	if !ok {
		return errors.New("unexpected data target")
	}
	*raw = json.RawMessage(r.data[req.OpName])
	return nil
}

func TestNewAPIWithClient(t *testing.T) {
	client := &recordingClient{data: map[string]string{
		"dashboardEnvironments": `{"environments":{"cursor":{"next":"c2"},"items":[{"id":"e1","name":"Dev","instances":[{"id":"i1","status":"PROVISIONED"}]}]}}`,
		"dashboardRuntimes":     `{"runtimes":{"cursor":{"next":null},"items":[{"id":"ecs","latest":{"version":"1.0.0"},"adoption":[]}]}}`,
		"dashboardApplications": `{"applications":{"cursor":{"next":null},"items":[{"id":"app","latest":{"version":"0.1.0","runtime":{"name":"ecs","version":"1.0.0"}}}]}}`,
	}}
	audit := &fakeAudit{page: types.Page[types.AuditLog]{Items: []types.AuditLog{{ID: "a1", Type: "deployment.completed"}}}}
	api := dashboard.NewAPIWithClient(client, audit, "acme")
	ctx := context.Background()

	envs, next, err := api.Environments(ctx, "c1")
	if err != nil || next != "c2" || len(envs) != 1 || envs[0].Instances[0].ID != "i1" {
		t.Fatalf("Environments = %v, %q, %v", envs, next, err)
	}
	vars, ok := client.reqs[0].Variables.(map[string]any)
	if !ok || vars["organizationId"] != "acme" {
		t.Fatalf("variables = %v", client.reqs[0].Variables)
	}
	if cursor, ok := vars["cursor"].(map[string]any); !ok || cursor["next"] != "c1" {
		t.Errorf("cursor = %v", vars["cursor"])
	}

	runtimes, next, err := api.Runtimes(ctx, "")
	if err != nil || next != "" || runtimes[0].ID != "ecs" {
		t.Errorf("Runtimes = %v, %q, %v", runtimes, next, err)
	}
	apps, _, err := api.Applications(ctx, "")
	if err != nil || apps[0].Latest.Runtime.Version != "1.0.0" {
		t.Errorf("Applications = %v, %v", apps, err)
	}

	since := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	events, err := api.Events(ctx, since)
	if err != nil || len(events) != 1 || events[0].ID != "a1" {
		t.Errorf("Events = %v, %v", events, err)
	}
	if !audit.input.TimeRangeStart.Equal(since) {
		t.Errorf("audit log read from %v, want %v", audit.input.TimeRangeStart, since)
	}
	audit.err = errors.New("forbidden")
	if _, err := api.Events(ctx, since); err == nil {
		t.Error("Events should report a failed read")
	}
}

type pagedAudit struct {
	pages map[string]types.Page[types.AuditLog]
}

func (p pagedAudit) ListPage(_ context.Context, input auditlogs.ListInput) (types.Page[types.AuditLog], error) {
	page, ok := p.pages[input.After]
	if !ok {
		return page, errors.New("unknown cursor")
	}
	return page, nil
}

func TestHistory(t *testing.T) {
	logs := func(ids ...string) []types.AuditLog {
		out := make([]types.AuditLog, 0, len(ids))
		for _, id := range ids {
			out = append(out, types.AuditLog{ID: id})
		}
		return out
	}
	tests := []struct {
		name    string
		pages   map[string]types.Page[types.AuditLog]
		want    []string
		wantErr bool
	}{
		{
			name: "every page, oldest first",
			pages: map[string]types.Page[types.AuditLog]{
				"":   {Items: logs("e4", "e3"), Next: "p2"},
				"p2": {Items: logs("e2", "e1")},
			},
			want: []string{"e1", "e2", "e3", "e4"},
		},
		{
			name:    "a failed page",
			pages:   map[string]types.Page[types.AuditLog]{"": {Items: logs("e2"), Next: "missing"}},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := dashboard.NewAPIWithClient(nil, pagedAudit{pages: tc.pages}, "acme")
			events, err := api.History(context.Background(), time.Time{})
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v", err)
			}
			got := make([]string, 0, len(events))
			for _, ev := range events {
				got = append(got, ev.ID)
			}
			if !tc.wantErr && !equal(got, tc.want) {
				t.Errorf("history = %v, want %v", got, tc.want)
			}
		})
	}
}

type fakeAudit struct {
	page  types.Page[types.AuditLog]
	input auditlogs.ListInput
	err   error
}

func (f *fakeAudit) ListPage(_ context.Context, input auditlogs.ListInput) (types.Page[types.AuditLog], error) {
	f.input = input
	return f.page, f.err
}

func TestToEvent(t *testing.T) {
	at := time.Date(2026, 11, 27, 7, 8, 44, 0, time.UTC)
	tests := []struct {
		name string
		log  types.AuditLog
		want dashboard.Event
	}{
		{
			name: "deployment by a system actor",
			log: types.AuditLog{
				ID: "1", OccurredAt: at, Type: "deployment.completed",
				Subject: "mri:acme:visits:dev:deployment:1a91",
				Actor:   &types.AuditLogActor{Type: "SYSTEM", Name: "system"},
				Data:    map[string]any{"entity": map[string]any{"action": "PROVISION", "deployed_by_name": "Automations"}},
			},
			want: dashboard.Event{ID: "1", At: at, Type: "deployment.completed", EnvID: "visits-dev", Actor: "Automations", Detail: "provision"},
		},
		{
			name: "named entity by a person",
			log: types.AuditLog{
				ID: "2", Type: "instance.created", Subject: "mri:acme:care:prod:instance:9",
				Actor: &types.AuditLogActor{Type: "ACCOUNT", Name: "ana@example.com"},
				Data:  map[string]any{"entity": map[string]any{"name": "db"}},
			},
			want: dashboard.Event{ID: "2", Type: "instance.created", EnvID: "care-prod", Actor: "ana@example.com", Detail: "db"},
		},
		{
			name: "outside any environment",
			log:  types.AuditLog{ID: "3", Type: "repo.created", Subject: "mri:acme:::repo:7"},
			want: dashboard.Event{ID: "3", Type: "repo.created"},
		},
		{
			name: "field of a remote reference",
			log: types.AuditLog{ID: "4", Type: "remote_reference.assigned", Subject: "mri:acme:auto:main:remote_reference:1",
				Data: map[string]any{"entity": map[string]any{"field": "vpc"}}},
			want: dashboard.Event{ID: "4", Type: "remote_reference.assigned", EnvID: "auto-main", Detail: "vpc"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := dashboard.ToEvent(tc.log); got != tc.want {
				t.Errorf("ToEvent = %+v, want %+v", got, tc.want)
			}
		})
	}
}
