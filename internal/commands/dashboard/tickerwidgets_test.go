package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/muesli/termenv"
)

const (
	teal   = "#5ED3B4"
	green  = "#6CCF97"
	orange = "#F59E42"
	red    = "#F25F5C"
	blue   = "#6C9EF0"
	gray   = "#7C8594"
)

// painted reports whether glyph is drawn in the color hex.
func painted(out, hex, glyph string) bool {
	open, _, _ := strings.Cut(lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("x"), "x")
	return strings.Contains(out, open+glyph)
}

func TestTickerWidget(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	if err := dashboard.UsePalette(dashboard.DefaultPalette); err != nil {
		t.Fatal(err)
	}
	app := inst("care-dev-app")
	set := func(e dashboard.StashEntry) dashboard.StashChange {
		return dashboard.StashChange{Scope: e.Scope, Key: e.Key, Entry: &e}
	}
	usage := func(value float64) dashboard.StashChange {
		return set(object(app, "md/v1/usage", dashboard.Values{"value": value, "max": 100.0, "label": "% CPU"}))
	}
	traffic := func(errs float64) dashboard.StashChange {
		return set(object(app, "md/v1/traffic", dashboard.Values{"rps": 40.0, "errors_5m": errs, "p95_ms": 180.0}))
	}
	src := func(scope, id, key string) map[string]any {
		return map[string]any{"source": "stash", "scope": map[string]any{scope: id}, "key": key}
	}
	pinned := []dashboard.Annotation{{
		Kind: "WIDGET", Surfaces: []string{dashboard.WidgetUsage}, Entity: dashboard.AnnotationEntity{Typename: "Instance", ID: "care-dev-app"},
		Inputs: dashboard.Values{"value": src("INSTANCE", "care-dev-app", "db/connections"), "max": src("instance", "care-dev-app", "db/max_connections")},
	}}
	pinnedStash := dashboard.Stash{app: {"db/max_connections": text(app, "db/max_connections", "50")}}
	tests := []struct {
		name        string
		change      dashboard.StashChange
		annotations []dashboard.Annotation
		stash       dashboard.Stash
		want        string
		color       string
		glyph       string
	}{
		{name: "health", change: set(text(app, "md/v1/health", "degraded")), want: "● degraded", color: orange, glyph: "●"},
		{name: "healthy", change: set(text(app, "md/v1/health", "OK")), want: "● OK", color: green, glyph: "●"},
		{name: "unknown health", change: set(text(app, "md/v1/health", "sleepy")), want: "● sleepy", color: gray, glyph: "●"},
		{name: "usage", change: usage(82), want: "▮▮▮▮▮▮▮▮▯▯ 82/100 % CPU", color: orange, glyph: "▮"},
		{name: "usage under 80%", change: usage(79), want: "▮▮▮▮▮▮▮▮▯▯ 79/100 % CPU", color: teal, glyph: "▮"},
		{name: "usage at 95%", change: usage(95), want: "▮▮▮▮▮▮▮▮▮▮ 95/100 % CPU", color: red, glyph: "▮"},
		{name: "usage past its max", change: usage(130), want: "▮▮▮▮▮▮▮▮▮▮ 130/100 % CPU", color: red, glyph: "▮"},
		{name: "usage with no max", change: set(object(app, "md/v1/usage", dashboard.Values{"value": 1.0, "max": 0.0})), want: "max: 0 · value: 1"},
		{name: "traffic", change: traffic(7), want: "40/s · ✕7 · 180ms", color: orange, glyph: "✕"},
		{name: "traffic with many errors", change: traffic(10), want: "40/s · ✕10 · 180ms", color: red, glyph: "✕"},
		{name: "traffic without errors", change: set(object(app, "md/v1/traffic", dashboard.Values{"rps": 1250.0, "errors_5m": 0.0})), want: "1.2k/s", color: teal, glyph: "1.2k/s"},
		{name: "banner", change: set(object(env("care-dev"), "md/v1/banner", dashboard.Values{"message": "Load test until 5pm", "level": "warning"})), want: "⚑ Load test until 5pm", color: orange, glyph: "⚑"},
		{name: "banner with no level", change: set(object(env("care-dev"), "md/v1/banner", dashboard.Values{"message": "Demo at 3pm"})), want: "⚑ Demo at 3pm", color: blue, glyph: "⚑"},
		{name: "links", change: set(object(env("care-dev"), "md/v1/links", dashboard.Values{"links": []any{
			map[string]any{"label": "Grafana", "url": "https://grafana.example.com"},
			map[string]any{"label": "", "url": "https://runbooks.example.com/care"},
			map[string]any{"label": "Bad", "url": "javascript:alert(1)"},
		}})), want: "🔗 Grafana, runbooks.example.com", color: teal, glyph: "🔗"},
		{name: "pinned widget", change: set(text(app, "db/connections", "37")), annotations: pinned, stash: pinnedStash, want: "▮▮▮▮▮▮▮▯▯▯ 37/50", color: teal, glyph: "▮"},
		{name: "pinned widget missing an input", change: set(text(app, "db/connections", "37")), annotations: pinned, want: "37"},
		{name: "well-known key in the wrong format", change: set(object(app, "md/v1/health", dashboard.Values{"state": "down"})), want: "state: down"},
		{name: "json", change: set(object(app, "deploy", dashboard.Values{
			"version": "1.2", "ok": true, "meta": map[string]any{"a": 1.0}, "tags": []any{"x"},
		})), want: "meta: {…} · ok: true · tags: […] · version: 1.2"},
		{name: "json that is not an object", change: set(dashboard.StashEntry{Scope: app, Key: "hosts", Value: ptr(`["a", {"b": 1}, 3, null]`), Format: "JSON"}), want: "a, {…}, 3, null"},
		{name: "text", change: set(text(app, "release", "3.2.1\nhotfix")), want: "3.2.1 hotfix"},
		{name: "deleted", change: dashboard.StashChange{Scope: app, Key: "release"}, want: "release"},
		{name: "clipped widget", change: set(object(env("care-dev"), "md/v1/banner", dashboard.Values{"message": strings.Repeat("freeze ", 20), "level": "error"})),
			want: "⚑ " + strings.Repeat("freeze ", 8) + "f…", color: red, glyph: "⚑"},
		{name: "clipped text", change: set(text(app, "note", strings.Repeat("word ", 20))), want: strings.Repeat("word ", 11) + "word…"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := dashboard.TickerWidget(tc.change, tc.annotations, tc.stash, fixedNow())
			if got := ansi.Strip(out); got != tc.want {
				t.Errorf("TickerWidget = %q, want %q", got, tc.want)
			}
			if w := ansi.StringWidth(out); w > 60 {
				t.Errorf("TickerWidget is %d cells wide, past the ticker's room", w)
			}
			if tc.color != "" && !painted(out, tc.color, tc.glyph) {
				t.Errorf("TickerWidget = %q, want %s in %s", out, tc.glyph, tc.color)
			}
		})
	}
}

func TestTickerWidgetStripsEscapes(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	app := inst("care-dev-app")
	evil := "\x1b]8;;https://evil\x07down\u202e"
	changes := []dashboard.StashEntry{
		text(app, "md/v1/health", evil),
		text(app, "note\x1b[2J", evil),
		object(env("care-dev"), "md/v1/banner", dashboard.Values{"message": evil}),
		object(env("care-dev"), "md/v1/links", dashboard.Values{"links": []any{map[string]any{"label": evil, "url": "https://example.com"}}}),
		object(app, "info", dashboard.Values{"k\x1b": evil}),
	}
	for _, e := range changes {
		out := dashboard.TickerWidget(dashboard.StashChange{Scope: e.Scope, Key: e.Key, Entry: &e}, nil, nil, fixedNow())
		if strings.ContainsAny(out, "\x1b\x07\u202e") {
			t.Errorf("TickerWidget(%q) = %q, want control characters stripped", e.Key, out)
		}
	}
}
