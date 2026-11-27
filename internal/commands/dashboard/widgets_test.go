package dashboard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func inst(id string) dashboard.Scope { return dashboard.Scope{Kind: "INSTANCE", ID: id} }
func env(id string) dashboard.Scope  { return dashboard.Scope{Kind: "ENVIRONMENT", ID: id} }

func text(scope dashboard.Scope, key, value string) dashboard.StashEntry {
	return dashboard.StashEntry{Scope: scope, Key: key, Value: value, Format: "TEXT", UpdatedAt: fixedNow().Format(time.RFC3339)}
}

func object(scope dashboard.Scope, key string, data dashboard.Values) dashboard.StashEntry {
	return dashboard.StashEntry{Scope: scope, Key: key, Value: "{}", Data: data, Format: "JSON", UpdatedAt: fixedNow().Format(time.RFC3339)}
}

func withStash() *fakeAPI {
	api := fixture()
	api.stash = []dashboard.StashEntry{
		text(inst("care-dev-app"), "md/health", "degraded"),
		object(inst("care-dev-app"), "md/usage", dashboard.Values{"value": 82.0, "max": 100.0, "label": "% CPU"}),
		object(inst("care-dev-queue"), "md/usage", dashboard.Values{"value": 1.0, "max": 0.0}),
		object(inst("platform-dev-db"), "md/traffic", dashboard.Values{"rps": 1250.0, "errors_5m": 7.0, "p95_ms": 180.0}),
		object(inst("care-dev-app"), "md/traffic", dashboard.Values{"rps": 40.0, "errors_5m": 0.0}),
		object(env("care-dev"), "md/banner", dashboard.Values{"message": "Load test until 5pm", "level": "warning"}),
		object(env("billing-dev"), "md/banner", dashboard.Values{"level": "error"}),
		object(env("care-dev"), "md/links", dashboard.Values{"links": []any{
			map[string]any{"label": "Grafana", "url": "https://grafana.example.com"},
			map[string]any{"label": "", "url": "https://skipped.example.com"},
		}}),
		text(inst("billing-dev-app"), "md/health", "ok"),
		object(inst("billing-dev-app"), "md/health", dashboard.Values{"state": "wrong format"}),
	}
	return api
}

func graphOf(t *testing.T, api *fakeAPI) *dashboard.Graph {
	t.Helper()
	snap := fixtureSnapshotFrom(t, api)
	st, err := dashboard.ReadStash(t.Context(), api, snap.Environments)
	if err != nil {
		t.Fatal(err)
	}
	g := dashboard.BuildGraph(snap)
	g.ApplyStash(st, snap.Annotations, fixedNow())
	return g
}

func fixtureSnapshotFrom(t *testing.T, api *fakeAPI) *dashboard.Snapshot {
	t.Helper()
	snap, err := dashboard.Fetch(t.Context(), api)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestWidgetsFromWellKnownKeys(t *testing.T) {
	g := graphOf(t, withStash())

	app := g.Instance("care-dev-app")
	if app.Health != "degraded" || app.Usage == nil || app.Usage.Value != 82 || app.Usage.Label != "% CPU" || app.Traffic.Rate != 40 {
		t.Errorf("care-dev-app = health %q usage %+v traffic %+v", app.Health, app.Usage, app.Traffic)
	}
	if g.Instance("care-dev-queue").Usage != nil {
		t.Error("a usage with no limit should be left out")
	}
	care := g.Env("care-dev")
	if care.Banner == nil || care.Banner.Level != "warning" || len(care.Links) != 1 || care.Links[0].Label != "Grafana" {
		t.Errorf("care-dev banner %+v links %+v", care.Banner, care.Links)
	}
	if g.Env("billing-dev").Banner != nil {
		t.Error("a banner without a message should be left out")
	}
	for _, l := range g.EnvLinks {
		tr := g.Traffic(l)
		switch l.From + ">" + l.To {
		case "platform-dev>care-dev":
			if tr == nil || tr.Rate != 1250 || tr.Errors != 7 || tr.Latency != 180 || !tr.HasLatency {
				t.Errorf("platform-dev>care-dev traffic = %+v", tr)
			}
		case "care-dev>billing-dev":
			if tr == nil || tr.Rate != 40 || tr.HasLatency {
				t.Errorf("care-dev>billing-dev traffic = %+v", tr)
			}
		default:
			if tr != nil {
				t.Errorf("%s>%s traffic = %+v, want none", l.From, l.To, tr)
			}
		}
	}
}

func TestWidgetsCannotEscapeTheTerminal(t *testing.T) {
	api := fixture()
	api.stash = []dashboard.StashEntry{
		object(env("care-dev"), "md/banner", dashboard.Values{"message": "fine\x1b]0;owned\x07\u009b2J"}),
		object(env("care-dev"), "md/links", dashboard.Values{"links": []any{
			map[string]any{"label": "ok\x1b[31m", "url": "https://ok.example.com\x1b\\"},
			map[string]any{"label": "local", "url": "file:///etc/passwd"},
			map[string]any{"label": "script", "url": "javascript:alert(1)"},
			map[string]any{"label": "Loud\u202eev\u200fil\u2028", "url": "HTTPS://LOUD.example.com"},
		}}),
	}
	care := graphOf(t, api).Env("care-dev")
	if care.Banner.Message != "fine]0;owned2J" {
		t.Errorf("banner = %q, want control characters stripped", care.Banner.Message)
	}
	if len(care.Links) != 2 || care.Links[0].Label != "ok[31m" || care.Links[0].URL != "https://ok.example.com\\" ||
		care.Links[1].Label != "Loudevil" {
		t.Errorf("links = %+v, want only the web links, cleaned", care.Links)
	}
}

func TestPinnedWidgets(t *testing.T) {
	src := func(scope, id, key string, path ...any) map[string]any {
		return map[string]any{"source": "stash", "scope": map[string]any{scope: id}, "key": key, "path": path}
	}
	api := withStash()
	api.stash = append(api.stash,
		text(inst("care-dev-queue"), "status/health", "down"),
		text(inst("care-dev-queue"), "db/connections", "37"),
		text(inst("care-dev-queue"), "db/max_connections", "100"),
		object(inst("care-dev-app"), "stats/traffic", dashboard.Values{"rps": 9.0, "errors_5m": 12.0}),
		text(env("care-dev"), "status/maintenance", "Database upgrade tonight"),
	)
	api.annotations = map[string][]dashboard.Annotation{"care": {
		{Kind: "WIDGET", Surfaces: []string{dashboard.WidgetHealth}, Entity: dashboard.AnnotationEntity{Typename: "Instance", ID: "care-dev-queue"},
			Inputs: dashboard.Values{"state": src("instance", "care-dev-queue", "status/health")}},
		{Kind: "WIDGET", Surfaces: []string{dashboard.WidgetUsage}, Entity: dashboard.AnnotationEntity{Typename: "Instance", ID: "care-dev-queue"},
			Inputs: dashboard.Values{"value": src("instance", "care-dev-queue", "db/connections"), "max": src("instance", "care-dev-queue", "db/max_connections")}},
		{Kind: "WIDGET", Surfaces: []string{dashboard.WidgetBanner}, Entity: dashboard.AnnotationEntity{Typename: "Environment", ID: "care-dev"},
			Inputs: dashboard.Values{"message": src("environment", "care-dev", "status/maintenance")}},
		{Kind: "WIDGET", Surfaces: []string{dashboard.WidgetHealth}, Entity: dashboard.AnnotationEntity{Typename: "Instance", ID: "care-dev-app"},
			Inputs: dashboard.Values{"state": src("instance", "care-dev-app", "missing")}},
		{Kind: "WIDGET", Surfaces: []string{dashboard.WidgetTraffic}, Entity: dashboard.AnnotationEntity{
			Typename: "Link", ID: "l1", FromComponent: ref("care-app", ""), ToComponent: ref("care-queue", ""),
		}, Inputs: dashboard.Values{
			"rate": src("instance", "care-dev-app", "stats/traffic", "rps"), "errors": src("instance", "care-dev-app", "stats/traffic", "errors_5m"),
		}},
	}}
	api.envPages[0][1].Instances[1].Component = ref("care-queue", "")
	g := graphOf(t, api)
	queue := g.Instance("care-dev-queue")
	if queue.Health != "down" || queue.Usage == nil || queue.Usage.Value != 37 || queue.Usage.Max != 100 {
		t.Errorf("care-dev-queue = health %q usage %+v; want widgets from pinned keys, with numbers read from text", queue.Health, queue.Usage)
	}
	if b := g.Env("care-dev").Banner; b == nil || b.Message != "Database upgrade tonight" {
		t.Errorf("banner = %+v; a pinned banner replaces the md/banner one", b)
	}
	if g.Instance("care-dev-app").Health != "degraded" {
		t.Error("a pinned widget that cannot resolve leaves the md/ widget in place")
	}
	if tr := g.Instance("care-dev-app").Traffic; tr == nil || tr.Rate != 9 || tr.Errors != 12 {
		t.Errorf("traffic = %+v; a pinned link widget draws on the instance the link leaves", tr)
	}
}

func TestWidgetsInTheView(t *testing.T) {
	m := press(loaded(t, withStash(), config(&opener{}), 200, 48), "/", "c", "a", "r", "e", "enter")
	out := view(m)
	for _, want := range []string{"⚑ Load test until 5pm", "↗ Grafana", "degraded", "82/100 % CPU", "40/s"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q:\n%s", want, out)
		}
	}
	if raw := m.View(); !strings.Contains(raw, "\x1b]8;;https://grafana.example.com") {
		t.Error("a header link should be a terminal hyperlink")
	}

	g := graphOf(t, withStash())
	pos := dashboard.Layout(g, 160, 48, dashboard.LayoutOptions{})
	scene := ansi.Strip(dashboard.Scene{Graph: g, Pos: pos, Frame: 5, Now: fixedNow()}.Render(160, 48))
	if !strings.Contains(scene, "1.2k/s 180ms ✕7") || !strings.Contains(scene, "40/s") {
		t.Errorf("links should carry their traffic:\n%s", scene)
	}
	stale := ansi.Strip(dashboard.Scene{Graph: g, Pos: pos, Frame: 5, Now: fixedNow().Add(time.Hour)}.Render(160, 48))
	if !strings.Contains(stale, "stale") {
		t.Errorf("traffic unreported for an hour should be stale:\n%s", stale)
	}
}

func TestRate(t *testing.T) {
	tests := []struct {
		n    float64
		want string
	}{{0, "0"}, {2.5, "2.5"}, {40, "40"}, {1250, "1.2k"}, {3_400_000, "3.4M"}}
	for _, tc := range tests {
		if got := dashboard.Rate(tc.n); got != tc.want {
			t.Errorf("Rate(%v) = %s, want %s", tc.n, got, tc.want)
		}
	}
}
