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
)

func TestReadStashScopes(t *testing.T) {
	api := fixture()
	big := make([]dashboard.Instance, 250)
	for i := range big {
		big[i] = dashboard.Instance{ID: fmt.Sprintf("big-dev-i%d", i), Status: "PROVISIONED"}
	}
	api.envPages = append(api.envPages, []dashboard.Environment{{ID: "big-dev", Name: "Dev", Project: ref("big", "Big"), Instances: big}})
	snap := fixtureSnapshotFrom(t, api)
	if _, err := dashboard.ReadStash(t.Context(), api, snap.Environments); err != nil {
		t.Fatal(err)
	}
	projects := map[string]int{}
	kinds := map[string]bool{}
	for _, read := range api.scopeReads {
		if len(read) > 200 {
			t.Errorf("a read asked for %d scopes, more than 200", len(read))
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
	for _, kind := range []string{"PROJECT", "ENVIRONMENT", "COMPONENT", "INSTANCE"} {
		if !kinds[kind] {
			t.Errorf("no %s stash was read", kind)
		}
	}
}

func TestReadStashKeepsTheReadsThatWork(t *testing.T) {
	api := withStash()
	api.failScope = "platform-dev"
	snap := fixtureSnapshotFrom(t, api)
	st, err := dashboard.ReadStash(t.Context(), api, snap.Environments)
	if err == nil {
		t.Error("a failed read should be reported")
	}
	if _, ok := st.Get(inst("care-dev-app"), "md/health", fixedNow()); !ok {
		t.Error("the projects that read should keep their entries")
	}
	if _, ok := st.Get(inst("platform-dev-db"), "md/traffic", fixedNow()); ok {
		t.Error("the failed project should have no entries")
	}
}

func TestStashExpiry(t *testing.T) {
	e := text(inst("care-dev-app"), "md/health", "down")
	e.ExpiresAt = fixedNow().Add(time.Minute).Format(time.RFC3339)
	st := dashboard.Stash{inst("care-dev-app"): {"md/health": e}}
	if _, ok := st.Get(inst("care-dev-app"), "md/health", fixedNow()); !ok {
		t.Error("an entry before it expires should read")
	}
	if _, ok := st.Get(inst("care-dev-app"), "md/health", fixedNow().Add(time.Hour)); ok {
		t.Error("an expired entry should not read")
	}
}

func TestStashChanges(t *testing.T) {
	health := text(inst("care-dev-app"), "md/health", "ok")
	usage := text(inst("care-dev-app"), "db/connections", "37")
	banner := text(env("care-dev"), "status/maintenance", "tonight")
	before := dashboard.Stash{
		inst("care-dev-app"): {"md/health": health, "db/connections": usage},
		env("care-dev"):      {"status/maintenance": banner},
	}
	changed := health
	changed.Value, changed.UpdatedAt = "down", fixedNow().Add(time.Second).Format(time.RFC3339)
	after := dashboard.Stash{
		inst("care-dev-app"): {"md/health": changed, "db/connections": usage, "release": text(inst("care-dev-app"), "release", "3.2.1")},
	}
	var got []string
	for _, c := range after.Changes(before) {
		state := "set"
		if c.Entry == nil {
			state = "deleted"
		}
		got = append(got, c.Scope.ID+" "+c.Key+" "+state)
	}
	want := []string{"care-dev status/maintenance deleted", "care-dev-app md/health set", "care-dev-app release set"}
	if !equal(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
}

func TestStashEvent(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	set := text(inst("care-dev-app"), "md/health", "down\nnow")
	tests := []struct {
		change dashboard.StashChange
		want   dashboard.Event
	}{
		{
			dashboard.StashChange{Scope: inst("care-dev-app"), Key: "md/health", Entry: &set},
			dashboard.Event{ID: "stash:INSTANCE:care-dev-app:md/health", At: fixedNow(), Type: "stash.set", EnvID: "care-dev", Detail: "md/health = down now"},
		},
		{
			dashboard.StashChange{Scope: dashboard.Scope{Kind: "PROJECT", ID: "care"}, Key: "release"},
			dashboard.Event{ID: "stash:PROJECT:care:release", At: fixedNow(), Type: "stash.deleted", Detail: "release"},
		},
	}
	for _, tc := range tests {
		if got := g.StashEvent(tc.change, fixedNow()); got != tc.want {
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
	api.stash[0] = text(inst("care-dev-app"), "md/health", "down")
	api.stash[0].UpdatedAt = fixedNow().Add(time.Second).Format(time.RFC3339)
	m = refreshed(t, m)
	out := view(m)
	if !strings.Contains(out, "stash set Care / Development · md/health = down") {
		t.Errorf("ticker should carry the change:\n%s", out)
	}
	if !strings.Contains(out, "md/health ") {
		t.Errorf("the environment should flash the key:\n%s", out)
	}
	api.stash[0].UpdatedAt = fixedNow().Add(2 * time.Second).Format(time.RFC3339)
	m = refreshed(t, m)
	ticker := ""
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.Contains(line, "LIVE") {
			ticker = line
		}
	}
	if strings.Count(ticker, "md/health") != 1 {
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
