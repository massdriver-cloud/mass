package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/muesli/termenv"
)

func TestThemes(t *testing.T) {
	if got := dashboard.Themes(); !equal(got, []string{"constellation", "blueprint", "rain"}) {
		t.Errorf("Themes = %v, want the default first", got)
	}
	if err := dashboard.UseTheme("disco"); err == nil || !strings.Contains(err.Error(), "constellation, blueprint, rain") {
		t.Errorf("an unknown theme should name the themes: %v", err)
	}
}

func TestThemesDraw(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(profile)
		if err := dashboard.UseTheme(dashboard.DefaultTheme); err != nil {
			t.Fatal(err)
		}
	})
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.Layout(g, 100, 30, dashboard.LayoutOptions{})
	frames := map[string]string{}
	for _, name := range dashboard.Themes() {
		if err := dashboard.UseTheme(name); err != nil {
			t.Fatal(err)
		}
		out := dashboard.Scene{Graph: g, Pos: pos, Frame: 9}.Render(100, 30)
		if !strings.Contains(ansi.Strip(out), "Development") {
			t.Errorf("%s drew no labels", name)
		}
		frames[name] = out
	}
	if frames["constellation"] == frames["rain"] || frames["rain"] == frames["blueprint"] {
		t.Error("each theme should look different")
	}
}
