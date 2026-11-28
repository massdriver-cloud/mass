package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/muesli/termenv"
)

func TestPalettes(t *testing.T) {
	if got := dashboard.Palettes(); !equal(got, []string{"night", "blueprint", "rain"}) {
		t.Errorf("Palettes = %v, want the default first", got)
	}
	if err := dashboard.UsePalette("disco"); err == nil || !strings.Contains(err.Error(), "night, blueprint, rain") {
		t.Errorf("an unknown palette should name the palettes: %v", err)
	}
}

func TestPaletteKey(t *testing.T) {
	t.Cleanup(func() { _ = dashboard.UsePalette(dashboard.DefaultPalette) })
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	for _, want := range []string{"blueprint", "rain", "night"} {
		m = press(m, "p")
		if dashboard.CurrentPalette() != want {
			t.Errorf("palette = %s, want %s", dashboard.CurrentPalette(), want)
		}
	}
}

func TestPalettesDraw(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(profile)
		if err := dashboard.UsePalette(dashboard.DefaultPalette); err != nil {
			t.Fatal(err)
		}
	})
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.Layout(g, 100, 30, dashboard.LayoutOptions{})
	frames := map[string]string{}
	for _, name := range dashboard.Palettes() {
		if err := dashboard.UsePalette(name); err != nil {
			t.Fatal(err)
		}
		out := dashboard.Scene{Graph: g, Pos: pos, Frame: 9}.Render(100, 30)
		if !strings.Contains(ansi.Strip(out), "Development") {
			t.Errorf("%s drew no labels", name)
		}
		frames[name] = out
	}
	if frames["night"] == frames["rain"] || frames["rain"] == frames["blueprint"] {
		t.Error("each palette should look different")
	}
}
