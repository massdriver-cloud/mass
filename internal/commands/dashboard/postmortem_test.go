package dashboard_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

const incidentFile = "incident-care-dev-app-latency-p99-20261126-0001"

func TestPostmortemOpens(t *testing.T) {
	d := &disk{}
	m := press(incidentScenario(t, d), "w")
	if len(d.files) > 0 {
		t.Errorf("opening the write-up should not write it, wrote %v", d.names())
	}
	out := view(m)
	for _, want := range []string{
		"WRITE-UP " + incidentFile + ".md",
		"Latency | p99 on care-dev-app",
		"Care / Development",
		"p99 over 2s",
		"Timeline",
		"deployment completed", "provision", "ana",
		"alarm fired", "deployment failed", "alarm cleared",
		"Root cause",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("write-up view is missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"# Latency", "\\|"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("write-up view should not show %q:\n%s", unwanted, out)
		}
	}
}

func TestPostmortemWrapsToTheBody(t *testing.T) {
	api := fixture()
	cfg := config(&opener{})
	cfg.WriteFile = (&disk{}).write
	m := loaded(t, api, cfg, 70, 60)
	setAlarm(careApp(api), "Latency", strings.Repeat("requests to checkout are slow ", 8)+"lastword")
	m = press(reread(t, m), "w")
	if out := view(m); !strings.Contains(out, "lastword") {
		t.Errorf("a long message should wrap, not be cut:\n%s", out)
	}
	m = incidentScenario(t, &disk{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 60})
	if out := view(press(next.(dashboard.Model), "w")); !strings.Contains(out, "│ ana") { //nolint:errcheck,forcetypeassert // Update always returns a Model
		t.Errorf("the timeline's last column should fit the body:\n%s", out)
	}
}

func TestPostmortemScrolls(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		want     []string
		unwanted []string
	}{
		{name: "opens at the top", want: []string{"Latency | p99 on care-dev-app", "Started"}, unwanted: []string{"Playback"}},
		{name: "G goes to the bottom", keys: []string{"G"}, want: []string{"Playback", "--playback"}, unwanted: []string{"Started"}},
		{name: "g goes back to the top", keys: []string{"G", "g"}, want: []string{"Started"}, unwanted: []string{"Playback"}},
		{name: "pgdown pages", keys: []string{"pgdown", "pgdown", "pgdown"}, want: []string{"Playback"}, unwanted: []string{"Started"}},
		{name: "pgup pages back", keys: []string{"G", "pgup", "pgup", "pgup"}, want: []string{"Started"}},
		{name: "j steps down", keys: []string{"j", "j", "j", "j", "j", "j"}, unwanted: []string{"Latency | p99 on"}},
		{name: "k steps back up", keys: []string{"j", "j", "j", "j", "j", "j", "k", "k", "k", "k", "k", "k"}, want: []string{"Latency | p99 on"}},
		{name: "arrows step too", keys: []string{"down", "down", "down", "down", "down", "down", "up", "up", "up", "up", "up", "up"}, want: []string{"Latency | p99 on"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := incidentScenario(t, &disk{})
			next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 22})
			m = press(next.(dashboard.Model), append([]string{"w"}, tc.keys...)...) //nolint:errcheck,forcetypeassert // Update always returns a Model
			out := view(m)
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			for _, unwanted := range tc.unwanted {
				if strings.Contains(out, unwanted) {
					t.Errorf("view should have scrolled past %q:\n%s", unwanted, out)
				}
			}
		})
	}
}

func TestPostmortemSaves(t *testing.T) {
	tests := []struct {
		name   string
		keys   []string
		prefix string
	}{
		{name: "unedited", prefix: "# Latency \\| p99 on care-dev-app\n"},
		{name: "while editing", keys: []string{"e", "Draft notes", "enter"}, prefix: "Draft notes\n# Latency"},
		{name: "after previewing", keys: []string{"e", "Draft notes", "enter", "esc"}, prefix: "Draft notes\n# Latency"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &disk{}
			m := save(t, press(incidentScenario(t, d), append([]string{"w"}, tc.keys...)...))
			if md := d.files[incidentFile+".md"]; !strings.HasPrefix(md, tc.prefix) || !strings.Contains(md, "## Root cause") {
				t.Errorf("saved write-up should start %q:\n%s", tc.prefix, md)
			}
			if rec := d.files[incidentFile+".json"]; !strings.Contains(rec, `"instanceId": "care-dev-app"`) {
				t.Errorf("save should write the recording too:\n%s", rec)
			}
			out := view(m)
			if !strings.Contains(out, "✎ wrote "+incidentFile+".md") {
				t.Errorf("save should name the file:\n%s", out)
			}
			if !strings.Contains(out, "WRITE-UP") || strings.Contains(out, "unsaved") {
				t.Errorf("the write-up should stay open, saved:\n%s", out)
			}
		})
	}
}

func TestPostmortemOverwritesAndReopens(t *testing.T) {
	d := &disk{}
	m := save(t, press(incidentScenario(t, d), "w", "e", "First pass", "enter"))
	m = save(t, press(m, "Second pass", "enter", "esc"))
	md := d.files[incidentFile+".md"]
	if !strings.HasPrefix(md, "First pass\nSecond pass\n# Latency") {
		t.Errorf("a second save should overwrite the first:\n%s", md)
	}
	out := view(press(m, "q", "w"))
	if !strings.Contains(out, "Second pass") || strings.Contains(out, "unsaved") {
		t.Errorf("reopening should show what was saved:\n%s", out)
	}
}

func TestPostmortemCloses(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		open    bool
		prompt  bool
		editing bool
	}{
		{name: "q closes", keys: []string{"q"}},
		{name: "esc closes", keys: []string{"esc"}},
		{name: "esc leaves the editor", keys: []string{"e", "esc"}, open: true},
		{name: "q types in the editor", keys: []string{"e", "q"}, open: true, editing: true},
		{name: "unsaved edits ask first", keys: []string{"e", "x", "esc", "q"}, open: true, prompt: true},
		{name: "esc asks too", keys: []string{"e", "x", "esc", "esc"}, open: true, prompt: true},
		{name: "y discards", keys: []string{"e", "x", "esc", "q", "y"}},
		{name: "n keeps them", keys: []string{"e", "x", "esc", "q", "n"}, open: true},
		{name: "edits undone close without asking", keys: []string{"e", "x", "backspace", "esc", "q"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := press(incidentScenario(t, &disk{}), append([]string{"w"}, tc.keys...)...)
			out := view(m)
			if got := strings.Contains(out, "WRITE-UP"); got != tc.open {
				t.Errorf("open = %v, want %v:\n%s", got, tc.open, out)
			}
			if got := strings.Contains(out, "discard changes? y/n"); got != tc.prompt {
				t.Errorf("prompt = %v, want %v:\n%s", got, tc.prompt, out)
			}
			if got := strings.Contains(out, "esc preview"); got != tc.editing {
				t.Errorf("editing = %v, want %v:\n%s", got, tc.editing, out)
			}
		})
	}
}

func TestPostmortemSaveAfterDiscardPrompt(t *testing.T) {
	d := &disk{}
	m := save(t, press(incidentScenario(t, d), "w", "e", "Kept", "esc", "q"))
	if !strings.HasPrefix(d.files[incidentFile+".md"], "Kept") {
		t.Errorf("ctrl+s at the prompt should save:\n%s", d.files[incidentFile+".md"])
	}
	if out := view(press(m, "q")); strings.Contains(out, "WRITE-UP") {
		t.Errorf("once saved, q should close:\n%s", out)
	}
}

func TestPostmortemHoldsTheKeys(t *testing.T) {
	m := press(incidentScenario(t, &disk{}), "w")
	for _, key := range []string{"2", "3", "/", "?", "n", "t", "f", "o", "1"} {
		next, cmd := pressCmd(m, key)
		if next.ActiveTab() != dashboard.TabConstellation || next.Selected() != m.Selected() {
			t.Errorf("%s should do nothing while the write-up is open", key)
		}
		if cmd != nil {
			t.Errorf("%s should not start anything while the write-up is open", key)
		}
		if out := view(next); !strings.Contains(out, "WRITE-UP") {
			t.Errorf("%s should leave the write-up open:\n%s", key, out)
		}
	}
}

func TestPostmortemMapKeepsUpdating(t *testing.T) {
	api := fixture()
	cfg := config(&opener{})
	cfg.WriteFile = (&disk{}).write
	m, read := pressCmd(loaded(t, api, cfg, 200, 48), "r")
	m = press(m, "3", "w")
	setAlarm(careApp(api), "Queue depth", "")
	next, _ := m.Update(read())
	m = next.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
	if out := view(m); !strings.Contains(out, "WRITE-UP") {
		t.Fatalf("a read should leave the write-up open:\n%s", out)
	}
	if out := view(press(m, "q")); !strings.Contains(out, "Queue depth") {
		t.Errorf("the read under the write-up should reach the map:\n%s", out)
	}
}

func TestPostmortemFooter(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		want     []string
		unwanted []string
	}{
		{name: "reading", want: []string{"↑↓ pgup pgdn scroll", "g G top/bottom", "e edit", "ctrl+s save", "q close"}, unwanted: []string{"1-5 tabs"}},
		{name: "editing", keys: []string{"e"}, want: []string{"ctrl+s save", "esc preview"}, unwanted: []string{"e edit", "q close"}},
		{name: "asking", keys: []string{"e", "x", "esc", "q"}, want: []string{"y discard", "n keep editing"}, unwanted: []string{"e edit"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := view(press(incidentScenario(t, &disk{}), append([]string{"w"}, tc.keys...)...))
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("footer is missing %q:\n%s", want, out)
				}
			}
			for _, unwanted := range tc.unwanted {
				if strings.Contains(out, unwanted) {
					t.Errorf("footer should not offer %q:\n%s", unwanted, out)
				}
			}
		})
	}
}
