package dashboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
	eventReads []dashboard.Log
	eventCalls int
	history    []dashboard.Event
	truncated  bool
	historyErr error
	since      time.Time
	widgets    []dashboard.Widget
	widgetErr  error
	failScope  string
	scopeMu    sync.Mutex
	scopeReads [][]dashboard.Scope
}

func (f *fakeAPI) Widgets(ctx context.Context, scopes []dashboard.Scope, _ string) ([]dashboard.Widget, string, error) {
	f.scopeMu.Lock()
	f.scopeReads = append(f.scopeReads, scopes)
	f.scopeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if f.widgetErr != nil {
		return nil, "", f.widgetErr
	}
	for _, s := range scopes {
		if s.ID == f.failScope {
			return nil, "", errors.New("read failed")
		}
	}
	var out []dashboard.Widget
	for _, w := range f.widgets {
		for _, s := range scopes {
			if s == w.Scope {
				out = append(out, w)
			}
		}
	}
	return out, "", nil
}

func (f *fakeAPI) History(_ context.Context, since time.Time) (dashboard.Log, error) {
	f.since = since
	return dashboard.Log{Events: f.history, Truncated: f.truncated}, f.historyErr
}

func (f *fakeAPI) Events(context.Context, time.Time) (dashboard.Log, error) {
	f.eventCalls++
	if f.eventCalls > len(f.eventReads) {
		return dashboard.Log{}, nil
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
		"dashboardWidgets": `{"alphaWidgetsForScopes":{"cursor":{"next":null},"items":[` +
			`{"name":"instance.status_dot","scope":{"kind":"INSTANCE","id":"i1"},"values":{"state":"down"}},` +
			`{"name":"instance.usage_bar","scope":{"kind":"INSTANCE","id":"i1"},"values":"{\"value\":3,\"max\":4}"}]}}`,
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
	if err != nil || len(events.Events) != 1 || events.Events[0].ID != "a1" {
		t.Errorf("Events = %v, %v", events, err)
	}
	if !audit.input.TimeRangeStart.Equal(since) {
		t.Errorf("audit log read from %v, want %v", audit.input.TimeRangeStart, since)
	}
	widgets, _, err := api.Widgets(ctx, []dashboard.Scope{{Kind: "ENVIRONMENT", ID: "e1"}, {Kind: "INSTANCE", ID: "i1"}}, "")
	if err != nil || len(widgets) != 2 || widgets[0].Values["state"] != "down" || widgets[1].Values["max"] != 4.0 {
		t.Errorf("Widgets = %+v, %v; want values read from an object or an encoded string", widgets, err)
	}
	wvars, _ := client.reqs[len(client.reqs)-1].Variables.(map[string]any)
	scopes, _ := wvars["scopes"].([]map[string]string)
	if len(scopes) != 2 || scopes[0]["environment"] != "e1" || scopes[1]["instance"] != "i1" {
		t.Errorf("scopes = %v, want one field per scope", wvars["scopes"])
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

func auditLogs(prefix string, from, to int) []types.AuditLog {
	out := make([]types.AuditLog, 0, from-to+1)
	for i := from; i >= to; i-- {
		out = append(out, types.AuditLog{ID: fmt.Sprintf("%s%d", prefix, i)})
	}
	return out
}

// newestFirst splits n audit log entries, newest first, into pages of 100.
func newestFirst(n int) map[string]types.Page[types.AuditLog] {
	pages := map[string]types.Page[types.AuditLog]{}
	for start, cursor := n, ""; start > 0; start -= 100 {
		next := ""
		if start > 100 {
			next = fmt.Sprintf("p%d", start-100)
		}
		pages[cursor] = types.Page[types.AuditLog]{Items: auditLogs("e", start, max(start-99, 1)), Next: next}
		cursor = next
	}
	return pages
}

func ids(events []dashboard.Event) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.ID)
	}
	return out
}

func TestReadAudit(t *testing.T) {
	tests := []struct {
		name          string
		pages         map[string]types.Page[types.AuditLog]
		wantEvents    int
		wantFirst     string
		wantLast      string
		wantTruncated bool
		wantErr       bool
	}{
		{name: "two pages, oldest first", pages: newestFirst(150), wantEvents: 150, wantFirst: "e1", wantLast: "e150"},
		{name: "more than a replay holds", pages: newestFirst(1200), wantEvents: 1000, wantFirst: "e201", wantLast: "e1200", wantTruncated: true},
		{name: "exactly a replay's worth", pages: newestFirst(1000), wantEvents: 1000, wantFirst: "e1", wantLast: "e1000"},
		{
			name:    "a failed page",
			pages:   map[string]types.Page[types.AuditLog]{"": {Items: auditLogs("e", 2, 2), Next: "missing"}},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := dashboard.NewAPIWithClient(nil, pagedAudit{pages: tc.pages}, "acme")
			log, err := api.History(context.Background(), time.Time{})
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v", err)
			}
			if tc.wantErr {
				return
			}
			got := ids(log.Events)
			if len(got) != tc.wantEvents || got[0] != tc.wantFirst || got[len(got)-1] != tc.wantLast || log.Truncated != tc.wantTruncated {
				t.Errorf("history = %d events %s..%s truncated %v, want %d %s..%s %v", len(got), got[0], got[len(got)-1],
					log.Truncated, tc.wantEvents, tc.wantFirst, tc.wantLast, tc.wantTruncated)
			}
		})
	}

	api := dashboard.NewAPIWithClient(nil, pagedAudit{pages: newestFirst(320)}, "acme")
	burst, err := api.Events(context.Background(), time.Time{})
	if err != nil || len(burst.Events) != 320 || burst.Truncated {
		t.Errorf("a burst of 320 events between polls read %d, %v; want every page", len(burst.Events), err)
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
