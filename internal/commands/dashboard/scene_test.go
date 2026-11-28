package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestSceneRender(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	const w, h = 100, 30
	pos := dashboard.Layout(g, w, h, dashboard.LayoutOptions{})
	tests := []struct {
		name  string
		scene dashboard.Scene
		want  []string
	}{
		{
			name:  "plain",
			scene: dashboard.Scene{Graph: g, Pos: pos},
			want:  []string{"Development", "Platform", "Template", "no access", "▲1"},
		},
		{
			name: "everything on",
			scene: dashboard.Scene{
				Graph: g, Pos: pos, Selected: "care-dev", Blast: g.BlastRadius("platform-dev-db"),
				ShowApps: true, GroupByProject: true, Filter: "care", Frame: 7,
			},
			want: []string{" Development ", "Care"},
		},
		{
			name: "unfolding with an event and cost",
			scene: dashboard.Scene{
				Graph: g, Pos: pos, ShowCost: true, Intro: 0.7, Frame: 3,
				Flashes: map[string]dashboard.Flash{"care-dev": {Age: 2, Label: "deployment completed"}, "gone": {Label: "x"}},
			},
			want: []string{"deployment completed", "Platform · $1.2k/mo", "↑1"},
		},
		{
			name:  "just starting",
			scene: dashboard.Scene{Graph: g, Pos: pos, Intro: 0.1},
			want:  []string{"Template"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.scene.Render(w, h)
			lines := strings.Split(out, "\n")
			if len(lines) != h {
				t.Fatalf("rendered %d lines, want %d", len(lines), h)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != w {
					t.Errorf("line %d is %d wide, want %d", i, got, w)
				}
			}
			plain := ansi.Strip(out)
			for _, want := range tc.want {
				if !strings.Contains(plain, want) {
					t.Errorf("render is missing %q", want)
				}
			}
		})
	}
}

func TestSceneMatches(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	tests := []struct {
		filter string
		env    string
		want   bool
	}{
		{"", "care-dev", true},
		{"CARE", "care-dev", true},
		{"queue", "care-dev", true},
		{"billing", "care-dev", false},
		{"sandboxes", "sandbox-demo", true},
	}
	for _, tc := range tests {
		if got := (dashboard.Scene{Filter: tc.filter}).Matches(g.Env(tc.env)); got != tc.want {
			t.Errorf("filter %q on %s = %v, want %v", tc.filter, tc.env, got, tc.want)
		}
	}
}

func TestSceneBlocks(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	cfg := config(&opener{})
	cfg.Render = dashboard.RenderBlocks
	blocks := loaded(t, fixture(), cfg, 160, 48)
	hasBraille := func(s string) bool {
		return strings.ContainsFunc(s, func(r rune) bool { return r > 0x2800 && r <= 0x28FF })
	}
	if !hasBraille(view(m)) {
		t.Error("the default render should draw braille")
	}
	if out := view(blocks); hasBraille(out) {
		t.Errorf("--render blocks should draw no braille:\n%s", out)
	}
}
