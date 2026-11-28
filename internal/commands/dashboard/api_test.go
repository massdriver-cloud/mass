package dashboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/auditlogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
)

type fakeAPI struct {
	envPages    [][]dashboard.Environment
	runtimes    []dashboard.Runtime
	apps        []dashboard.Application
	err         error
	recent      string
	noRecent    bool
	dirErr      error
	envReads    [][]string
	eventReads  []dashboard.Log
	eventCalls  int
	history     []dashboard.Event
	truncated   bool
	historyErr  error
	since       time.Time
	stash       []dashboard.StashEntry
	stashErr    error
	failScope   string
	scopeMu     sync.Mutex
	scopeReads  [][]dashboard.Scope
	annotations map[string][]dashboard.Annotation
	annErr      error
	cancelled   int
	// streamed holds the events each environment's first subscription sends
	// before it ends; with streamed nil, subscribing needs a personal token.
	streamed   map[string][]dashboard.StashEvent
	subscribed []string
	// logReads are each deployment's log reads in turn, the last repeating.
	logReads map[string][]dashboard.DeploymentLog
	logCalls map[string]int
	// liveLogs holds what each deployment's stream sends before it ends; with
	// liveLogs nil, streaming needs a personal token.
	liveLogs   map[string][]dashboard.LogBatch
	logStreams []context.Context
	approved   []string
	deployed   []string
	actionErr  error
	// deploys are every deployment the API knows, in any order.
	deploys     []dashboard.Deployment
	deployReads []dashboard.DeploymentQuery
	deployErr   error
	peeks       map[string]dashboard.DeploymentPeek
	peekErr     error
	rejected    []string
	reasons     []string
}

func (f *fakeAPI) ApproveDeployment(_ context.Context, deploymentID string) error {
	f.approved = append(f.approved, deploymentID)
	return f.actionErr
}

func (f *fakeAPI) Deploy(_ context.Context, instanceID string) error {
	f.deployed = append(f.deployed, instanceID)
	return f.actionErr
}

func (f *fakeAPI) SubscribeStash(_ context.Context, projectID, environmentID string) (*streaming.Stream[dashboard.StashEvent], error) {
	f.scopeMu.Lock()
	defer f.scopeMu.Unlock()
	if f.streamed == nil || slices.Contains(f.subscribed, projectID+"/"+environmentID) {
		return nil, streaming.ErrRequiresPAT
	}
	f.subscribed = append(f.subscribed, projectID+"/"+environmentID)
	events := f.streamed[environmentID]
	c := make(chan dashboard.StashEvent, len(events))
	for _, ev := range events {
		c <- ev
	}
	close(c)
	return streaming.New(c, nil), nil
}

func (f *fakeAPI) StashEntries(ctx context.Context, scopes []dashboard.Scope, _ string) ([]dashboard.StashEntry, string, error) {
	f.scopeMu.Lock()
	f.scopeReads = append(f.scopeReads, scopes)
	f.scopeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if f.stashErr != nil {
		return nil, "", f.stashErr
	}
	for _, s := range scopes {
		if s.ID == f.failScope {
			return nil, "", errors.New("read failed")
		}
	}
	var out []dashboard.StashEntry
	for _, e := range f.stash {
		for _, s := range scopes {
			if s == e.Scope {
				out = append(out, e)
			}
		}
	}
	return out, "", nil
}

func (f *fakeAPI) Annotations(_ context.Context, projectID, _ string) ([]dashboard.Annotation, string, error) {
	if f.annErr != nil {
		return nil, "", f.annErr
	}
	return f.annotations[projectID], "", nil
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

func (f *fakeAPI) EnvironmentsByID(ctx context.Context, ids []string, _ string) ([]dashboard.Environment, string, error) {
	f.scopeMu.Lock()
	defer f.scopeMu.Unlock()
	f.envReads = append(f.envReads, ids)
	if err := ctx.Err(); err != nil {
		f.cancelled++
		return nil, "", err
	}
	if f.err != nil {
		return nil, "", f.err
	}
	var out []dashboard.Environment
	for _, page := range f.envPages {
		for _, env := range page {
			if slices.Contains(ids, env.ID) {
				out = append(out, env)
			}
		}
	}
	return out, "", nil
}

func (f *fakeAPI) Directory(context.Context, string) ([]dashboard.Place, string, error) {
	if f.dirErr != nil {
		return nil, "", f.dirErr
	}
	var out []dashboard.Place
	for _, page := range f.envPages {
		for _, env := range page {
			out = append(out, dashboard.Place{ID: env.ID, Name: env.Name, Project: env.Project})
		}
	}
	return out, "", nil
}

func (f *fakeAPI) RecentEnvironment(context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.noRecent {
		return "", nil
	}
	if f.recent != "" {
		return f.recent, nil
	}
	return "care-dev", nil
}

// all loads every environment of the fixture.
func (f *fakeAPI) all() []dashboard.Environment {
	var out []dashboard.Environment
	for _, page := range f.envPages {
		out = append(out, page...)
	}
	return out
}

func usedBy(instanceID, envID string) dashboard.InstanceResource {
	return dashboard.InstanceResource{Resource: dashboard.ProducedResource{RemoteReferences: dashboard.RemoteReferences{
		Items: []dashboard.RemoteReference{{Instance: &dashboard.ResourceInstance{ID: instanceID, Environment: ref(envID, "")}}},
	}}}
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
		{ID: "platform-dev-db", Status: "PROVISIONED", Alarms: firing, Cost: usd(1250), Component: ref("platform-db", ""),
			Resources: []dashboard.InstanceResource{usedBy("care-dev-app", "care-dev")}, Dependencies: []dashboard.Dependency{dependsOn("platform-dev-network", "platform-dev")}},
	}}
	care := dashboard.Environment{ID: "care-dev", Name: "Development", Project: ref("care", "Care"), Instances: []dashboard.Instance{
		{ID: "care-dev-app", Status: "PROVISIONED", Bundle: appBundle("care-app", "0.1.0", "1.0.0"), Component: ref("care-app", ""),
			Resources:       []dashboard.InstanceResource{usedBy("billing-dev-app", "billing-dev")},
			DeployedVersion: "0.1.0", AvailableUpgrade: "0.2.0",
			Dependencies: []dashboard.Dependency{dependsOn("platform-dev-db", "platform-dev"), dependsOn("secret-vpc", "secret-prod")}},
		{ID: "care-dev-queue", Status: "INITIALIZED", AvailableUpgrade: "1.0.0",
			LatestDeployment: &dashboard.Deployment{ID: "dep-1", Status: "PROPOSED", Action: "PROVISION"}},
	}}
	billing := dashboard.Environment{ID: "billing-dev", Name: "Development", Project: ref("billing", "Billing"), Instances: []dashboard.Instance{
		{ID: "billing-dev-app", Status: "FAILED", Bundle: appBundle("billing-app", "2.0.0", "1.0.0"), Component: ref("billing-app", ""),
			LatestDeployment: &dashboard.Deployment{ID: "dep-billing", Status: "RUNNING", Action: "PROVISION", CreatedAt: "2026-11-25T23:58:00Z"},
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

// fixtureSnapshot is the whole fixture, every environment loaded.
func fixtureSnapshot(t *testing.T) *dashboard.Snapshot {
	t.Helper()
	api := fixture()
	snap, err := dashboard.Fetch(context.Background(), api, dashboard.Where{Focus: "care-dev"})
	if err != nil {
		t.Fatal(err)
	}
	snap.Environments, snap.Missing = api.all(), []string{"secret-prod"}
	return snap
}

func TestFetchCleansPublisherText(t *testing.T) {
	api := fixture()
	api.runtimes[0].ID = "ecs\x1b[2J"
	api.runtimes[0].Adoption = []dashboard.Adoption{{Version: "1.0.0\u202e"}}
	api.apps[0].ID = "care-app\x07"
	api.apps[0].Latest.Runtime.Name = "ecs\x1b[2J"
	billing := &api.envPages[0][2].Instances[0]
	billing.DeployedVersion = "1.9.0\u200f"
	billing.LatestProvision.Bundle.Name = "billing-app\x1b]0;x\x07"
	billing.LatestProvision.Bundle.Runtime.Name = "ecs\x1b[2J"

	snap, err := dashboard.Fetch(context.Background(), api, dashboard.Where{Focus: "care-dev"})
	if err != nil {
		t.Fatal(err)
	}
	g := dashboard.BuildGraph(snap)
	inst := g.Instance("billing-dev-app")
	tests := []struct{ name, got, want string }{
		{"runtime", snap.Runtimes[0].ID, "ecs[2J"},
		{"adoption", snap.Runtimes[0].Adoption[0].Version, "1.0.0"},
		{"application", snap.Applications[0].ID, "care-app"},
		{"application runtime", snap.Applications[0].Latest.Runtime.Name, "ecs[2J"},
		{"deployed", inst.Deployed, "1.9.0"},
		{"bundle", inst.Bundle, "billing-app]0;x"},
		{"instance runtime", inst.Runtime, "ecs[2J"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
	if r := dashboard.BuildRollout(g, snap, snap.Runtimes[0], dashboard.BumpNone); len(r.Instances) != 1 || len(r.Apps) != 1 {
		t.Errorf("rollout = %d instances, %d apps; cleaned names should still join", len(r.Instances), len(r.Apps))
	}
}

func TestFetchAroundFocus(t *testing.T) {
	tests := []struct {
		name        string
		where       dashboard.Where
		recent      string
		wantFocus   string
		wantEnvs    []string
		wantMissing []string
		wantErr     string
	}{
		{
			name: "the most recent environment and its neighbors", wantFocus: "care-dev",
			wantEnvs: []string{"care-dev", "platform-dev", "billing-dev"}, wantMissing: []string{"secret-prod"},
		},
		{
			name: "a consumer found through the resources it uses", where: dashboard.Where{Focus: "platform-dev"}, wantFocus: "platform-dev",
			wantEnvs: []string{"platform-dev", "care-dev"},
		},
		{
			name: "two hops out", where: dashboard.Where{Focus: "billing-dev", Depth: 2}, wantFocus: "billing-dev",
			wantEnvs: []string{"billing-dev", "care-dev", "platform-dev"}, wantMissing: []string{"secret-prod"},
		},
		{
			name: "a fork and its parent", where: dashboard.Where{Focus: "sandbox-demo"}, wantFocus: "sandbox-demo",
			wantEnvs: []string{"sandbox-demo", "sandbox-template"},
		},
		{name: "an environment that is not there", where: dashboard.Where{Focus: "nope"}, wantErr: `environment "nope": not found`},
		{name: "an organization with nothing deployed", recent: "-", wantErr: "no environments to show"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := fixture()
			if tc.recent == "-" {
				api.envPages = nil
				api.recent = ""
				api.noRecent = true
			}
			snap, err := dashboard.Fetch(context.Background(), api, tc.where)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, env := range snap.Environments {
				got = append(got, env.ID)
			}
			if snap.Focus != tc.wantFocus || !equal(got, tc.wantEnvs) || !equal(snap.Missing, tc.wantMissing) {
				t.Errorf("focus %s, envs %v, missing %v; want %s, %v, %v", snap.Focus, got, snap.Missing, tc.wantFocus, tc.wantEnvs, tc.wantMissing)
			}
		})
	}
	if _, err := dashboard.Fetch(context.Background(), &fakeAPI{err: errors.New("boom")}, dashboard.Where{}); err == nil {
		t.Error("a failed read should be reported")
	}
}

func TestFindPlace(t *testing.T) {
	places := []dashboard.Place{
		{ID: "billing-dev", Name: "Development", Project: ref("billing", "Billing")},
		{ID: "care-dev", Name: "Development", Project: ref("care", "Care")},
		{ID: "care-prod", Name: "Production", Project: ref("care", "Care")},
	}
	tests := map[string]string{"care-prod": "care-prod", "care": "care-dev", "production": "care-prod", "billing development": "", "  ": ""}
	for query, want := range tests {
		got, ok := dashboard.FindPlace(places, query)
		if (want == "") == ok || got.ID != want {
			t.Errorf("FindPlace(%q) = %q, %v; want %q", query, got.ID, ok, want)
		}
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
		"dashboardDirectory":    `{"environments":{"cursor":{"next":null},"items":[{"id":"e1","name":"Dev","project":{"id":"p","name":"P"}}]}}`,
		"dashboardRecent":       `{"deployments":{"items":[{"instance":null},{"instance":{"id":"i9","environment":{"id":"e9"}}}]},"environments":{"items":[{"id":"e1"}]}}`,
	}}
	audit := &fakeAudit{page: types.Page[types.AuditLog]{Items: []types.AuditLog{{ID: "a1", Type: "deployment.completed"}}}}
	api := dashboard.NewAPIWithClient(client, dashboard.Services{Audit: audit}, "acme")
	ctx := context.Background()

	envs, next, err := api.EnvironmentsByID(ctx, []string{"e1"}, "c1")
	if err != nil || next != "c2" || len(envs) != 1 || envs[0].Instances[0].ID != "i1" {
		t.Fatalf("EnvironmentsByID = %v, %q, %v", envs, next, err)
	}
	vars, ok := client.reqs[0].Variables.(map[string]any)
	if !ok || vars["organizationId"] != "acme" || !equal(vars["ids"].([]string), []string{"e1"}) { //nolint:errcheck,forcetypeassert // the request sets ids
		t.Fatalf("variables = %v", client.reqs[0].Variables)
	}
	if cursor, ok := vars["cursor"].(map[string]any); !ok || cursor["next"] != "c1" {
		t.Errorf("cursor = %v", vars["cursor"])
	}
	places, _, err := api.Directory(ctx, "")
	if err != nil || len(places) != 1 || places[0].Project.Name != "P" {
		t.Errorf("Directory = %v, %v", places, err)
	}
	if recent, recentErr := api.RecentEnvironment(ctx); recentErr != nil || recent != "e9" {
		t.Errorf("RecentEnvironment = %q, %v; want the environment of the newest deployment", recent, recentErr)
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
	audit.err = errors.New("forbidden")
	if _, err := api.Events(ctx, since); err == nil {
		t.Error("Events should report a failed read")
	}
}

type fakeStash struct {
	scopes []stash.Scope
	input  stash.ForScopesInput
	page   types.Page[stash.Entry]
}

func (f *fakeStash) Subscribe(context.Context, string, string) (*streaming.Stream[stash.Event], error) {
	return nil, streaming.ErrRequiresPAT
}

func (f *fakeStash) ListForScopes(_ context.Context, scopes []stash.Scope, input stash.ForScopesInput) (types.Page[stash.Entry], error) {
	f.scopes, f.input = scopes, input
	return f.page, nil
}

func TestNewAPIWithClientStash(t *testing.T) {
	client := &recordingClient{data: map[string]string{
		"dashboardAnnotations": `{"alphaAnnotations":{"cursor":{"next":null},"items":[` +
			`{"id":"a1","kind":"NOTE","surfaces":["environment.canvas_legend"],"template":"hi","inputs":null,"entity":{"__typename":"Environment","id":"e1"}}]}}`,
	}}
	down := "down"
	st := &fakeStash{page: types.Page[stash.Entry]{Items: []stash.Entry{{Scope: stash.Scope{Kind: stash.ScopeInstance, ID: "i1"}, Key: "md/v1/health", Value: &down}}, Next: "c2"}}
	api := dashboard.NewAPIWithClient(client, dashboard.Services{Audit: &fakeAudit{}, Stash: st}, "acme")
	ctx := context.Background()
	scopes := []dashboard.Scope{{Kind: stash.ScopeEnvironment, ID: "e1"}, {Kind: stash.ScopeInstance, ID: "i1"}}
	entries, next, err := api.StashEntries(ctx, scopes, "c1")
	if err != nil || next != "c2" || len(entries) != 1 || *entries[0].Value != "down" {
		t.Errorf("StashEntries = %+v, %q, %v", entries, next, err)
	}
	if len(st.scopes) != 2 || st.input.After != "c1" || st.input.PageSize == 0 {
		t.Errorf("read scopes %v with %+v, want both scopes from the cursor", st.scopes, st.input)
	}
	notes, _, err := api.Annotations(ctx, "p1", "")
	if err != nil || len(notes) != 1 || notes[0].Entity.Typename != "Environment" || notes[0].Template != "hi" {
		t.Errorf("Annotations = %+v, %v", notes, err)
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
			api := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: pagedAudit{pages: tc.pages}}, "acme")
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

	api := dashboard.NewAPIWithClient(nil, dashboard.Services{Audit: pagedAudit{pages: newestFirst(320)}}, "acme")
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

// costlyClient refuses any page over most items, and any request that is not
// paged, as the API refuses a request whose complexity is over its maximum.
type costlyClient struct {
	most   int
	limits []int
}

func (c *costlyClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	vars, _ := req.Variables.(map[string]any)
	cursor, paged := vars["cursor"].(map[string]any)
	limit, _ := cursor["limit"].(int)
	c.limits = append(c.limits, limit)
	if !paged || limit > c.most {
		return fmt.Errorf("input:1:1: Operation %s is too complex: complexity is %d and maximum is %d", req.OpName, limit*1000, c.most*1000)
	}
	raw, _ := resp.Data.(*json.RawMessage)
	*raw = json.RawMessage(`{"environments":{"cursor":{"next":null},"items":[{"id":"e1"}]}}`)
	return nil
}

func TestTooComplexAsksForSmallerPages(t *testing.T) {
	environments := func(api dashboard.API) error {
		_, _, err := api.EnvironmentsByID(context.Background(), []string{"e1"}, "")
		return err
	}
	recent := func(api dashboard.API) error {
		_, err := api.RecentEnvironment(context.Background())
		return err
	}
	tests := map[string]struct {
		read       func(dashboard.API) error
		most       int
		wantLimits []int
		wantErr    bool
	}{
		"fits":                     {read: environments, most: 50, wantLimits: []int{50, 50}},
		"retries a smaller page":   {read: environments, most: 20, wantLimits: []int{50, 18, 18}},
		"too complex for one item": {read: environments, most: 0, wantLimits: []int{50, 1}, wantErr: true},
		"unpaged is not retried":   {read: recent, most: 50, wantLimits: []int{0}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &costlyClient{most: tc.most}
			api := dashboard.NewAPIWithClient(client, dashboard.Services{Audit: &fakeAudit{}}, "acme")
			err := tc.read(api)
			if err == nil {
				err = tc.read(api)
			}
			if tc.wantErr != errors.Is(err, dashboard.ErrTooComplex) {
				t.Fatalf("err = %v", err)
			}
			if !slices.Equal(client.limits, tc.wantLimits) {
				t.Errorf("limits = %v, want %v; the second read should start at the page size that fit", client.limits, tc.wantLimits)
			}
		})
	}
}
