package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func inst(id string) dashboard.Scope { return dashboard.Scope{Kind: "INSTANCE", ID: id} }
func env(id string) dashboard.Scope  { return dashboard.Scope{Kind: "ENVIRONMENT", ID: id} }

func withWidgets() *fakeAPI {
	api := fixture()
	api.widgets = []dashboard.Widget{
		{Name: dashboard.WidgetHealth, Scope: inst("care-dev-app"), Values: dashboard.Values{"state": "degraded"}},
		{Name: dashboard.WidgetUsage, Scope: inst("care-dev-app"), Values: dashboard.Values{"value": 82.0, "max": 100.0, "label": "% CPU"}},
		{Name: dashboard.WidgetUsage, Scope: inst("care-dev-queue"), Values: dashboard.Values{"value": 1.0, "max": 0.0}},
		{Name: dashboard.WidgetTraffic, Scope: inst("platform-dev-db"), Values: dashboard.Values{"rate": 1250.0, "errors": 7.0}},
		{Name: dashboard.WidgetTraffic, Scope: inst("care-dev-app"), Values: dashboard.Values{"rate": 40.0}},
		{Name: dashboard.WidgetBanner, Scope: env("care-dev"), Values: dashboard.Values{"message": "Load test until 5pm", "level": "warning"}},
		{Name: dashboard.WidgetBanner, Scope: env("billing-dev"), Values: dashboard.Values{"level": "error"}},
		{Name: dashboard.WidgetLinks, Scope: env("care-dev"), Values: dashboard.Values{"links": []any{
			map[string]any{"label": "Grafana", "url": "https://grafana.example.com"},
			map[string]any{"label": "", "url": "https://skipped.example.com"},
		}}},
		{Name: dashboard.WidgetHealth, Scope: inst("nobody"), Values: dashboard.Values{"state": "down"}},
	}
	return api
}

func TestWidgetsOnTheGraph(t *testing.T) {
	api := withWidgets()
	snap, err := dashboard.Fetch(context.Background(), api)
	if err != nil {
		t.Fatal(err)
	}
	g := dashboard.BuildGraph(snap)

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
			if tr == nil || tr.Rate != 1250 || tr.Errors != 7 {
				t.Errorf("platform-dev>care-dev traffic = %+v", tr)
			}
		case "care-dev>billing-dev":
			if tr == nil || tr.Rate != 40 {
				t.Errorf("care-dev>billing-dev traffic = %+v", tr)
			}
		default:
			if tr != nil {
				t.Errorf("%s>%s traffic = %+v, want none", l.From, l.To, tr)
			}
		}
	}
}

func TestWidgetScopes(t *testing.T) {
	api := fixture()
	big := make([]dashboard.Instance, 250)
	for i := range big {
		big[i] = dashboard.Instance{ID: fmt.Sprintf("big-dev-i%d", i), Status: "PROVISIONED"}
	}
	api.envPages = append(api.envPages, []dashboard.Environment{{ID: "big-dev", Name: "Dev", Project: ref("big", "Big"), Instances: big}})
	if _, err := dashboard.Fetch(context.Background(), api); err != nil {
		t.Fatal(err)
	}
	projects := map[string]int{}
	for _, read := range api.scopeReads {
		if len(read) > 200 {
			t.Errorf("a read asked for %d scopes, more than 200", len(read))
		}
		seen := map[string]bool{}
		for _, s := range read {
			seen[strings.SplitN(s.ID, "-", 2)[0]] = true
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
}

func TestWidgetsKeepTheReadsThatWork(t *testing.T) {
	api := withWidgets()
	api.failScope = "platform-dev"
	g := dashboard.BuildGraph(fixtureSnapshotFrom(t, api))
	if g.Instance("care-dev-app").Health != "degraded" || g.Instance("platform-dev-db").Traffic != nil {
		t.Error("a failed read for one project should leave out only that project's widgets")
	}
}

func TestWidgetsCannotEscapeTheTerminal(t *testing.T) {
	api := fixture()
	api.widgets = []dashboard.Widget{
		{Name: dashboard.WidgetBanner, Scope: env("care-dev"), Values: dashboard.Values{"message": "fine\x1b]0;owned\x07\u009b2J"}},
		{Name: dashboard.WidgetLinks, Scope: env("care-dev"), Values: dashboard.Values{"links": []any{
			map[string]any{"label": "ok\x1b[31m", "url": "https://ok.example.com\x1b\\"},
			map[string]any{"label": "local", "url": "file:///etc/passwd"},
			map[string]any{"label": "script", "url": "javascript:alert(1)"},
			map[string]any{"label": "Loud\u202eevil", "url": "HTTPS://LOUD.example.com"},
		}}},
	}
	g := dashboard.BuildGraph(fixtureSnapshotFrom(t, api))
	care := g.Env("care-dev")
	if care.Banner.Message != "fine]0;owned2J" {
		t.Errorf("banner = %q, want control characters stripped", care.Banner.Message)
	}
	if len(care.Links) != 2 || care.Links[0].Label != "ok[31m" || care.Links[0].URL != "https://ok.example.com\\" ||
		care.Links[1].Label != "Loudevil" {
		t.Errorf("links = %+v, want only the web link, cleaned", care.Links)
	}
}

func TestWidgetsFailQuietly(t *testing.T) {
	api := withWidgets()
	api.widgetErr = errors.New(`Cannot query field "alphaWidgetsForScopes"`)
	snap, err := dashboard.Fetch(context.Background(), api)
	if err != nil || len(snap.Widgets) != 0 || len(snap.Environments) != 5 {
		t.Errorf("Fetch = %d widgets, %d environments, %v; want the dashboard without widgets", len(snap.Widgets), len(snap.Environments), err)
	}
}

func TestWidgetsInTheView(t *testing.T) {
	m := press(loaded(t, withWidgets(), config(&opener{}), 200, 48), "/", "c", "a", "r", "e", "enter")
	out := view(m)
	for _, want := range []string{
		"⚑ Load test until 5pm", "↗ Grafana", "degraded", "82/100 % CPU", "40/s",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q:\n%s", want, out)
		}
	}
	if raw := m.View(); !strings.Contains(raw, "\x1b]8;;https://grafana.example.com") {
		t.Error("a header link should be a terminal hyperlink")
	}

	g := dashboard.BuildGraph(fixtureSnapshotFrom(t, withWidgets()))
	pos := dashboard.Layout(g, 160, 48, dashboard.LayoutOptions{})
	scene := ansi.Strip(dashboard.Scene{Graph: g, Pos: pos, Frame: 5}.Render(160, 48))
	if !strings.Contains(scene, "1.2k/s ✕7") || !strings.Contains(scene, "40/s") {
		t.Errorf("links should carry their traffic:\n%s", scene)
	}
}

func fixtureSnapshotFrom(t *testing.T, api *fakeAPI) *dashboard.Snapshot {
	t.Helper()
	snap, err := dashboard.Fetch(context.Background(), api)
	if err != nil {
		t.Fatal(err)
	}
	return snap
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
