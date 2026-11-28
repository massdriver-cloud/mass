package dashboard_test

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestThemesList(t *testing.T) {
	if got := dashboard.Themes(); !equal(got, []string{"constellation", "metro", "radar"}) {
		t.Errorf("Themes = %v", got)
	}
	if got := dashboard.NextTheme("radar"); got != "constellation" {
		t.Errorf("NextTheme(radar) = %s, want it to wrap", got)
	}
}

func TestMetroLayout(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.MetroLayout(g, 160, 48)
	if len(pos) != len(g.Envs) {
		t.Fatalf("placed %d of %d", len(pos), len(g.Envs))
	}
	if !(pos["platform-dev"].X < pos["care-dev"].X && pos["care-dev"].X < pos["billing-dev"].X) {
		t.Errorf("resources should flow left to right: platform %v, care %v, billing %v",
			pos["platform-dev"], pos["care-dev"], pos["billing-dev"])
	}
	if demo, template := pos["sandbox-demo"], pos["sandbox-template"]; demo.X != template.X || demo.Y == template.Y {
		t.Errorf("a project's environments in one layer should stack in its band: demo %v, template %v", demo, template)
	}
}

func TestRadarLayout(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.RadarLayout(g, 160, 48, "care-dev")
	center := pos["care-dev"]
	if center.X != 160 || center.Y != 96 {
		t.Errorf("focus at %v, want the middle of the scope", center)
	}
	dist := func(id string) float64 {
		p := pos[id]
		return math.Hypot((p.X-center.X)/136, (p.Y-center.Y)/88)
	}
	if !(dist("platform-dev") < dist("sandbox-demo") && math.Abs(dist("platform-dev")-dist("billing-dev")) < 0.01) {
		t.Errorf("one hop out should share a ring inside the unreachable edge: platform %.2f billing %.2f sandbox %.2f",
			dist("platform-dev"), dist("billing-dev"), dist("sandbox-demo"))
	}
}

func TestThemesRender(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	for _, theme := range dashboard.Themes() {
		pos := dashboard.LayoutTheme(theme, g, 160, 48, dashboard.LayoutOptions{Center: "care-dev"})
		out := ansi.Strip(dashboard.Scene{Graph: g, Pos: pos, Theme: theme, Selected: "care-dev", Frame: 7}.Render(160, 48))
		for _, want := range []string{"Development", "Template"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s is missing %q:\n%s", theme, want, out)
			}
		}
		if theme == dashboard.ThemeMetro && !strings.ContainsAny(out, "━┃┏┓┗┛") {
			t.Errorf("metro should draw track:\n%s", out)
		}
	}
}

func TestThemeKey(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	for _, want := range []string{"metro", "radar", "constellation"} {
		m = press(m, "v")
		if m.Theme() != want {
			t.Errorf("theme = %s, want %s", m.Theme(), want)
		}
	}
}

func TestBannerStrip(t *testing.T) {
	out := view(loaded(t, withStash(), config(&opener{}), 160, 48))
	if !strings.Contains(out, "⚑ Care / Development · Load test until 5pm") {
		t.Errorf("a banner in view should run across the top of the map:\n%s", out)
	}
}
