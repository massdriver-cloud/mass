package dashboard_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
)

func TestReadStashScopes(t *testing.T) {
	api := fixture()
	big := make([]dashboard.Instance, 250)
	for i := range big {
		big[i] = dashboard.Instance{ID: fmt.Sprintf("big-dev-i%d", i), Status: "PROVISIONED"}
	}
	api.envPages = append(api.envPages, []dashboard.Environment{{ID: "big-dev", Name: "Dev", Project: ref("big", "Big"), Instances: big}})
	snap := fixtureSnapshotFrom(t, api)
	if err := dashboard.ReadStash(t.Context(), api, snap.Environments).Err; err != nil {
		t.Fatal(err)
	}
	projects := map[string]int{}
	kinds := map[stash.ScopeKind]bool{}
	for _, read := range api.scopeReads {
		if len(read) > stash.MaxScopes {
			t.Errorf("a read asked for %d scopes, more than %d", len(read), stash.MaxScopes)
		}
		seen := map[string]bool{}
		for _, s := range read {
			seen[strings.SplitN(s.ID, "-", 2)[0]] = true
			kinds[s.Kind] = true
		}
		if len(seen) != 1 {
			t.Errorf("a read mixed projects: %v", seen)
		}
		for p := range seen {
			projects[p]++
		}
	}
	if projects["big"] != 2 || projects["care"] != 1 {
		t.Errorf("reads per project = %v, want big split in two", projects)
	}
	for _, kind := range []stash.ScopeKind{stash.ScopeProject, stash.ScopeEnvironment, stash.ScopeComponent, stash.ScopeInstance} {
		if !kinds[kind] {
			t.Errorf("no %s stash was read", kind)
		}
	}
}

func TestReadStashKeepsTheReadsThatWork(t *testing.T) {
	api := withStash()
	api.failScope = "platform-dev"
	snap := fixtureSnapshotFrom(t, api)
	read := dashboard.ReadStash(t.Context(), api, snap.Environments)
	st, err := read.Stash, read.Err
	if err == nil {
		t.Error("a failed read should be reported")
	}
	if _, ok := st.Get(inst("care-dev-app"), "md/v1/health", fixedNow()); !ok {
		t.Error("the projects that read should keep their entries")
	}
	if _, ok := st.Get(inst("platform-dev-db"), "md/v1/traffic", fixedNow()); ok {
		t.Error("the failed project should have no entries")
	}
	if len(read.Failed) == 0 || read.Failed[0].ID != "platform" {
		t.Errorf("failed scopes = %v, want the platform project's", read.Failed)
	}
}

func TestStashExpiry(t *testing.T) {
	e := text(inst("care-dev-app"), "md/v1/health", "down")
	e.ExpiresAt = fixedNow().Add(time.Minute)
	st := dashboard.Stash{inst("care-dev-app"): {"md/v1/health": e}}
	if _, ok := st.Get(inst("care-dev-app"), "md/v1/health", fixedNow()); !ok {
		t.Error("an entry before it expires should read")
	}
	if _, ok := st.Get(inst("care-dev-app"), "md/v1/health", fixedNow().Add(time.Hour)); ok {
		t.Error("an expired entry should not read")
	}
}

func TestStashChanges(t *testing.T) {
	health := text(inst("care-dev-app"), "md/v1/health", "ok")
	usage := text(inst("care-dev-app"), "db/connections", "37")
	banner := text(env("care-dev"), "status/maintenance", "tonight")
	before := dashboard.Stash{
		inst("care-dev-app"): {"md/v1/health": health, "db/connections": usage},
		env("care-dev"):      {"status/maintenance": banner},
	}
	changed := health
	down := "down"
	changed.Value, changed.UpdatedAt = &down, fixedNow().Add(time.Second)
	after := dashboard.Stash{
		inst("care-dev-app"): {"md/v1/health": changed, "db/connections": usage, "release": text(inst("care-dev-app"), "release", "3.2.1")},
	}
	var got []string
	for _, c := range after.Changes(before) {
		state := "set"
		if c.Entry == nil {
			state = "deleted"
		}
		got = append(got, c.Scope.ID+" "+c.Key+" "+state)
	}
	want := []string{"care-dev status/maintenance deleted", "care-dev-app md/v1/health set", "care-dev-app release set"}
	if !equal(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
}

func TestStashEvent(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	set := text(inst("care-dev-app"), "md/v1/health", "down\nnow")
	tests := []struct {
		change dashboard.StashChange
		want   dashboard.Event
	}{
		{
			dashboard.StashChange{Scope: inst("care-dev-app"), Key: "md/v1/health", Entry: &set},
			dashboard.Event{ID: "stash:instance:care-dev-app:md/v1/health", At: fixedNow(), Type: "stash.set", EnvID: "care-dev", Detail: "md/v1/health ● down now"},
		},
		{
			dashboard.StashChange{Scope: dashboard.Scope{Kind: stash.ScopeProject, ID: "care"}, Key: "release"},
			dashboard.Event{ID: "stash:project:care:release", At: fixedNow(), Type: "stash.deleted", Detail: "release"},
		},
	}
	for _, tc := range tests {
		got := g.StashEvent(tc.change, nil, nil, fixedNow())
		if ansi.Strip(got.Widget) != got.Detail {
			t.Errorf("widget %q should read as its detail %q", got.Widget, got.Detail)
		}
		if got.Widget = ""; got != tc.want {
			t.Errorf("StashEvent = %+v, want %+v", got, tc.want)
		}
	}
}

// refreshed re-reads the organization and the stash, feeding every message
// back, as bubbletea would.
func refreshed(t *testing.T, m dashboard.Model) dashboard.Model {
	t.Helper()
	next, cmd := pressCmd(m, "r")
	var model tea.Model = next
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		if msg == nil {
			continue
		}
		var more tea.Cmd
		model, more = model.Update(msg)
		queue = append(queue, more)
	}
	return model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
}

func TestStashFeed(t *testing.T) {
	api := withStash()
	m := loaded(t, api, config(&opener{}), 200, 48)
	if out := view(m); strings.Contains(out, "stash set") {
		t.Fatalf("the first read of the stash is not news:\n%s", out)
	}
	api.stash[0] = text(inst("care-dev-app"), "md/v1/health", "down")
	api.stash[0].UpdatedAt = fixedNow().Add(time.Second)
	m = refreshed(t, m)
	out := view(m)
	if !strings.Contains(out, "stash set Care / Development · md/v1/health ● down") {
		t.Errorf("ticker should carry the change:\n%s", out)
	}
	if !strings.Contains(out, "md/v1/health ") {
		t.Errorf("the environment should flash the key:\n%s", out)
	}
	api.stash[0].UpdatedAt = fixedNow().Add(2 * time.Second)
	m = refreshed(t, m)
	ticker := ""
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.Contains(line, "LIVE") {
			ticker = line
		}
	}
	if strings.Count(ticker, "md/v1/health") != 1 {
		t.Errorf("a key set again should replace its line on the ticker:\n%s", ticker)
	}
}

func TestStashFeedSurvivesAFailedRead(t *testing.T) {
	api := withStash()
	m := press(loaded(t, api, config(&opener{}), 200, 48), "/", "c", "a", "r", "e", "enter")
	api.stashErr = errors.New("boom")
	if out := view(refreshed(t, m)); !strings.Contains(out, "degraded") {
		t.Errorf("a failed stash read should keep the widgets already drawn:\n%s", out)
	}
}

func TestStashApply(t *testing.T) {
	down := text(inst("care-dev-app"), "md/v1/health", "down")
	before := dashboard.Stash{inst("care-dev-app"): {"md/v1/health": text(inst("care-dev-app"), "md/v1/health", "ok"), "owner": text(inst("care-dev-app"), "owner", "visits")}}
	after := before.Apply([]dashboard.StashEvent{
		{Action: stash.EventSet, Scope: inst("care-dev-app"), Key: "md/v1/health", Entry: &down},
		{Action: stash.EventDeleted, Scope: inst("care-dev-app"), Key: "owner"},
		{Action: stash.EventSet, Scope: env("care-dev"), Key: "md/v1/banner", Entry: ptr(text(env("care-dev"), "md/v1/banner", "{}"))},
	})
	if e, _ := after.Get(inst("care-dev-app"), "md/v1/health", fixedNow()); e.Value == nil || *e.Value != "down" {
		t.Errorf("md/v1/health = %+v, want the set value", e)
	}
	if _, ok := after.Get(inst("care-dev-app"), "owner", fixedNow()); ok {
		t.Error("a deleted key should be gone")
	}
	if _, ok := after.Get(env("care-dev"), "md/v1/banner", fixedNow()); !ok {
		t.Error("a key in a scope with no entries yet should be added")
	}
	if e, _ := before.Get(inst("care-dev-app"), "md/v1/health", fixedNow()); *e.Value != "ok" || len(before) != 1 {
		t.Error("Apply should leave the stash it was called on as it was")
	}
}

func TestStashKeepTouched(t *testing.T) {
	current := dashboard.Stash{inst("care-dev-app"): {"md/v1/health": text(inst("care-dev-app"), "md/v1/health", "down")}}
	touched := map[dashboard.Scope]map[string]time.Time{inst("care-dev-app"): {
		"md/v1/health": fixedNow().Add(time.Second), "owner": fixedNow().Add(time.Second), "old": fixedNow().Add(-time.Second),
	}}
	read := dashboard.Stash{inst("care-dev-app"): {
		"md/v1/health": text(inst("care-dev-app"), "md/v1/health", "ok"),
		"owner":        text(inst("care-dev-app"), "owner", "visits"),
		"old":          text(inst("care-dev-app"), "old", "kept"),
	}}
	read.KeepTouched(current, touched, fixedNow())
	if e, _ := read.Get(inst("care-dev-app"), "md/v1/health", fixedNow()); *e.Value != "down" {
		t.Errorf("md/v1/health = %s; a key streamed after the read started should keep the streamed value", *e.Value)
	}
	if _, ok := read.Get(inst("care-dev-app"), "owner", fixedNow()); ok {
		t.Error("a key deleted after the read started should stay deleted")
	}
	if _, ok := read.Get(inst("care-dev-app"), "old", fixedNow()); !ok {
		t.Error("a key streamed before the read started should take the read's value")
	}
}

func ptr[T any](v T) *T { return &v }

func TestStashFeedShowsEmoji(t *testing.T) {
	api := withStash()
	cfg := config(&opener{})
	cfg.Flashes = dashboard.FlashEmoji
	m := loaded(t, api, cfg, 200, 48)
	api.stash[0] = text(inst("care-dev-app"), "md/v1/health", "down")
	api.stash[0].UpdatedAt = fixedNow().Add(time.Second)
	out := view(refreshed(t, m))
	if !strings.Contains(out, "📌 Care / Development · md/v1/health ● down") || strings.Contains(out, "stash set") {
		t.Errorf("the ticker should lead a stash change with 📌, not words:\n%s", out)
	}
}
