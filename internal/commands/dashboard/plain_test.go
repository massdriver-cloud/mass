package dashboard_test

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestPlainTheme(t *testing.T) {
	cfg := config(&opener{})
	cfg.Theme = dashboard.ThemePlain
	m := loaded(t, fixture(), cfg, 160, 48)
	out := view(m)
	for _, want := range []string{"Platform", "Care", "Billing", "▲ 1 alarming", "⟳ 1 deploying", "↑ 1", "1/2 up"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plain view is missing %q:\n%s", want, out)
		}
	}
	if strings.ContainsFunc(out, func(r rune) bool { return r >= 0x2800 && r <= 0x28FF }) {
		t.Errorf("the plain view should draw no braille:\n%s", out)
	}
	m = press(m, "down")
	if m.Selected() != "platform-dev" {
		t.Errorf("selected %q; down should move to the next line, Platform's", m.Selected())
	}
}

func manyEnvs(n int) *dashboard.Graph {
	envs := make([]dashboard.Environment, n)
	for i := range envs {
		envs[i] = dashboard.Environment{ID: fmt.Sprintf("e%02d", i), Name: fmt.Sprintf("env %02d", i), Project: ref(fmt.Sprintf("p%d", i%4), fmt.Sprintf("Project %d", i%4))}
	}
	return dashboard.BuildGraph(&dashboard.Snapshot{Environments: envs})
}

func TestPlainLayout(t *testing.T) {
	tests := map[string]struct {
		envs, w, h int
		placed     int
		more       bool
	}{
		"fits in one column":    {envs: 8, w: 60, h: 40, placed: 8},
		"spills into a column":  {envs: 40, w: 140, h: 30, placed: 40},
		"too many for the room": {envs: 40, w: 60, h: 20, placed: 14, more: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := manyEnvs(tc.envs)
			pos := dashboard.PlainLayout(g, tc.w, tc.h)
			if len(pos) != tc.placed {
				t.Errorf("placed %d of %d, want %d", len(pos), tc.envs, tc.placed)
			}
			for id, p := range pos {
				if p.Y >= float64(tc.h*4) || p.X >= float64(tc.w*2) {
					t.Errorf("%s at %+v is off the %dx%d canvas", id, p, tc.w, tc.h)
				}
			}
			out := ansi.Strip(dashboard.Scene{Graph: g, Pos: pos, Theme: dashboard.ThemePlain}.Render(tc.w, tc.h))
			if strings.Contains(out, "more · / finds") != tc.more {
				t.Errorf("a +N more line = %v, want %v:\n%s", !tc.more, tc.more, out)
			}
		})
	}
}

func TestPlainExport(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	scene := dashboard.Scene{Graph: g, Pos: dashboard.PlainLayout(g, 140, 40), Theme: dashboard.ThemePlain, Selected: "care-dev"}
	out := dashboard.ExportSVG(scene, 140, 40, "acme")
	dec := xml.NewDecoder(strings.NewReader(out))
	for {
		if _, err := dec.Token(); err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("invalid XML: %v", err)
			}
			break
		}
	}
	for _, want := range []string{"Platform", "Care", "▲ 1 alarming", "↑ 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plain export is missing %q", want)
		}
	}
	if strings.Contains(out, "<path") {
		t.Error("the plain export should draw no links")
	}
}
