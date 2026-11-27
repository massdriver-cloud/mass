package dashboard_test

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestCanvasDots(t *testing.T) {
	tests := []struct {
		name string
		dots [][2]int
		want string
	}{
		{name: "top left", dots: [][2]int{{0, 0}}, want: "⠁ "},
		{name: "bottom right of first cell", dots: [][2]int{{1, 3}}, want: "⢀ "},
		{name: "full cell", dots: [][2]int{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 0}, {1, 1}, {1, 2}, {1, 3}}, want: "⣿ "},
		{name: "second cell", dots: [][2]int{{2, 0}}, want: " ⠁"},
		{name: "off canvas", dots: [][2]int{{-1, 0}, {9, 9}}, want: "  "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := dashboard.NewCanvas(2, 1)
			for _, d := range tc.dots {
				c.Dot(d[0], d[1], lipgloss.Color("#fff"), 0)
			}
			if got := ansi.Strip(c.Render()); got != tc.want {
				t.Errorf("render = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCanvasText(t *testing.T) {
	c := dashboard.NewCanvas(9, 2)
	c.Line(0, 0, 17, 7, lipgloss.Color("#fff"), 0, 0)
	c.TextCentered(4, 1, "hello", lipgloss.NewStyle())
	lines := strings.Split(ansi.Strip(c.Render()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "hello") {
		t.Fatalf("render = %q", lines)
	}
	if strings.TrimSpace(lines[0]) == "" {
		t.Error("the line left no dots on the first row")
	}
	for _, line := range lines {
		if w := lipgloss.Width(line); w != 9 {
			t.Errorf("line %q is %d wide, want 9", line, w)
		}
	}
}

func TestCanvasShapes(t *testing.T) {
	c := dashboard.NewCanvas(10, 5)
	white := lipgloss.Color("#fff")
	c.Ring(dashboard.Point{X: 10, Y: 10}, 4, func(float64) lipgloss.Color { return white }, 0)
	c.Disc(dashboard.Point{X: 10, Y: 10}, 1, white, 0)
	c.Curve(dashboard.Point{X: 0, Y: 0}, dashboard.Point{X: 10, Y: 0}, dashboard.Point{X: 19, Y: 19}, dashboard.Stroke{Color: white, Dash: 2, Gap: 1, Until: 0.8})
	if got := ansi.Strip(c.Render()); strings.TrimSpace(strings.ReplaceAll(got, "\n", "")) == "" {
		t.Error("shapes drew nothing")
	}
}

func TestBezier(t *testing.T) {
	a, b := dashboard.Point{X: 0, Y: 0}, dashboard.Point{X: 10, Y: 0}
	ctrl := dashboard.Bend(a, b, 0.5)
	if ctrl.X != 5 || ctrl.Y != 5 {
		t.Errorf("bend = %+v, want {5 5}", ctrl)
	}
	tests := []struct {
		t    float64
		want dashboard.Point
	}{
		{0, a},
		{1, b},
		{0.5, dashboard.Point{X: 5, Y: 2.5}},
	}
	for _, tc := range tests {
		got := dashboard.Bezier(a, ctrl, b, tc.t)
		if math.Abs(got.X-tc.want.X) > 1e-9 || math.Abs(got.Y-tc.want.Y) > 1e-9 {
			t.Errorf("bezier(%v) = %+v, want %+v", tc.t, got, tc.want)
		}
	}
}

func TestCanvasStyles(t *testing.T) {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f00"))
	tests := map[string]func(c *dashboard.Canvas){
		"text":         func(c *dashboard.Canvas) { c.Text(0, 0, "hi", style) },
		"put":          func(c *dashboard.Canvas) { id := c.Style(style); c.Put(0, 0, "h", id); c.Put(1, 0, "i", id) },
		"two in a row": func(c *dashboard.Canvas) { c.Text(0, 0, "h", style); c.Text(1, 0, "i", style) },
	}
	for name, draw := range tests {
		t.Run(name, func(t *testing.T) {
			c := dashboard.NewCanvas(2, 1)
			draw(c)
			if got, want := c.Render(), style.Render("hi"); ansi.Strip(got) != "hi" || (name != "two in a row" && got != want) {
				t.Errorf("render = %q, want %q", got, want)
			}
		})
	}
}
