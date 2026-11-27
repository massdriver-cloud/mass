package dashboard_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

var fixedNow = func() time.Time { return time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC) }

type fakeLinks struct{}

func (fakeLinks) Environment(id string) string { return "https://app/env/" + id }
func (fakeLinks) Instance(id string) string    { return "https://app/inst/" + id }
func (fakeLinks) Runtime(id string) string     { return "https://app/rt/" + id }

type opener struct{ urls []string }

func (o *opener) open(url string) error {
	o.urls = append(o.urls, url)
	if strings.Contains(url, "rt/") {
		return errors.New("no browser")
	}
	return nil
}

func config(o *opener) dashboard.Config {
	return dashboard.Config{Org: "acme", EventPoll: time.Millisecond, Now: fixedNow, Links: fakeLinks{}, Open: o.open}
}

// pump runs commands and feeds their messages back, as bubbletea would, for
// a number of steps.
func pump(t *testing.T, m tea.Model, queue []tea.Cmd, steps int) dashboard.Model {
	t.Helper()
	for i := 0; i < steps && len(queue) > 0; i++ {
		cmd := queue[0]
		queue = queue[1:]
		if cmd == nil {
			continue
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		var next tea.Cmd
		m, next = m.Update(msg)
		queue = append(queue, next)
	}
	model, ok := m.(dashboard.Model)
	if !ok {
		t.Fatalf("model is %T", m)
	}
	return model
}

// loaded reads the organization and the first events, and sizes the screen.
func loaded(t *testing.T, api dashboard.API, cfg dashboard.Config, w, h int) dashboard.Model {
	t.Helper()
	var m tea.Model = dashboard.NewModel(api, cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	model := pump(t, m, []tea.Cmd{m.Init()}, 4)
	return model
}

func press(m dashboard.Model, keys ...string) dashboard.Model {
	next, _ := pressCmd(m, keys...)
	return next
}

func pressCmd(m dashboard.Model, keys ...string) (dashboard.Model, tea.Cmd) {
	named := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "backspace": tea.KeyBackspace,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
	}
	var cmd tea.Cmd
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if kt, ok := named[k]; ok {
			msg = tea.KeyMsg{Type: kt}
		}
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
	}
	return m, cmd
}

func view(m dashboard.Model) string {
	return strings.Join(strings.Fields(ansi.Strip(m.View())), " ")
}

func TestModelLoads(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 200, 48)
	if m.Selected() != "platform-dev" {
		t.Errorf("selected = %q, want the environment with the newest firing alarm", m.Selected())
	}
	out := view(m)
	for _, want := range []string{
		"acme", "1 Constellation", "5 environments · 4 projects", "⟳ 1 deploying", "↑ 1", "$1.2k/mo", "1 alarming",
		"BLAST RADIUS", "b blast ●", "LIVE", "waiting for activity", "o open",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q:\n%s", want, out)
		}
	}
}

func TestModelKeys(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		wantTab      dashboard.Tab
		wantSelected string
		wantView     []string
	}{
		{name: "runtimes tab", keys: []string{"2"}, wantTab: dashboard.TabRuntimes, wantView: []string{"RUNTIMES", "ecs latest 1.0.0"}},
		{name: "what if major", keys: []string{"2", "M"}, wantTab: dashboard.TabRuntimes, wantView: []string{"Publishing 2.0.0 leaves 2 of 2"}},
		{name: "alarms tab", keys: []string{"3"}, wantTab: dashboard.TabAlarms, wantView: []string{"FIRING ALARMS (1)", "CPU high", "2 env · 2 inst downstream"}},
		{name: "alarm to blast", keys: []string{"1", "tab", "3", "enter"}, wantTab: dashboard.TabConstellation, wantSelected: "platform-dev"},
		{name: "upgrades tab", keys: []string{"4"}, wantTab: dashboard.TabUpgrades, wantView: []string{"UPGRADES AVAILABLE (1)", "care-dev-app", "care-app", "0.1.0 → 0.2.0"}},
		{name: "upgrade to map", keys: []string{"4", "down", "enter"}, wantTab: dashboard.TabConstellation, wantSelected: "care-dev"},
		{name: "find", keys: []string{"/", "b", "i", "l", "enter"}, wantSelected: "billing-dev"},
		{name: "find then clear", keys: []string{"/", "z", "z", "backspace", "esc"}, wantSelected: "platform-dev"},
		{
			name: "toggles", keys: []string{"b", "a", "c", "g", "s"},
			wantView: []string{"b blast ○", "a apps ○", "c cost ●", "g projects ●", "s producers on top ●"},
		},
		{name: "legend", keys: []string{"?"}, wantView: []string{"LEGEND", "Blast radius, source to downstream", "Deploying", "Upgrade available"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := press(loaded(t, fixture(), config(&opener{}), 200, 48), tc.keys...)
			if m.ActiveTab() != tc.wantTab {
				t.Errorf("tab = %d, want %d", m.ActiveTab(), tc.wantTab)
			}
			if tc.wantSelected != "" && m.Selected() != tc.wantSelected {
				t.Errorf("selected = %q, want %q", m.Selected(), tc.wantSelected)
			}
			out := view(m)
			for _, want := range tc.wantView {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestModelOpens(t *testing.T) {
	tests := []struct {
		name      string
		keys      []string
		wantURL   string
		wantToast string
	}{
		{name: "environment", keys: []string{"o"}, wantURL: "https://app/env/platform-dev", wantToast: "opened https://app/env/platform-dev"},
		{name: "alarm", keys: []string{"3", "o"}, wantURL: "https://app/inst/platform-dev-db", wantToast: "opened"},
		{name: "upgrade", keys: []string{"4", "o"}, wantURL: "https://app/inst/care-dev-app", wantToast: "opened"},
		{name: "runtime without a browser", keys: []string{"2", "o"}, wantURL: "https://app/rt/ecs", wantToast: "could not open a browser: https://app/rt/ecs"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &opener{}
			m, cmd := pressCmd(loaded(t, fixture(), config(o), 200, 48), tc.keys...)
			if cmd == nil {
				t.Fatal("o should open the selection")
			}
			next, _ := m.Update(cmd())
			if len(o.urls) != 1 || o.urls[0] != tc.wantURL {
				t.Errorf("opened %v, want %s", o.urls, tc.wantURL)
			}
			if out := view(next.(dashboard.Model)); !strings.Contains(out, tc.wantToast) { //nolint:errcheck,forcetypeassert // Update always returns a Model
				t.Errorf("footer is missing %q:\n%s", tc.wantToast, out)
			}
		})
	}
	if _, cmd := pressCmd(loaded(t, fixture(), dashboard.Config{Now: fixedNow}, 200, 48), "o"); cmd != nil {
		t.Error("without links there is nothing to open")
	}
}

func TestModelEvents(t *testing.T) {
	api := fixture()
	newest := dashboard.Event{ID: "new", Type: "deployment.completed", EnvID: "billing-dev", At: fixedNow(), Detail: "provision", Actor: "ana"}
	api.eventReads = []dashboard.Log{
		{Events: []dashboard.Event{{ID: "old", Type: "instance.created", EnvID: "care-dev", At: fixedNow().Add(-time.Hour / 2)}}},
		{Events: []dashboard.Event{{ID: "old", Type: "instance.created", EnvID: "care-dev", At: fixedNow().Add(-time.Hour / 2)}, newest}},
		{Events: []dashboard.Event{newest, {ID: "burst", Type: "instance.updated", EnvID: "care-dev", At: fixedNow()}}, Truncated: true},
	}
	var m tea.Model = dashboard.NewModel(api, config(&opener{}))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 48})
	model := pump(t, m, []tea.Cmd{m.Init()}, 40)

	if api.eventCalls < 3 {
		t.Fatalf("read events %d times, want polls after the first read", api.eventCalls)
	}
	if got := len(api.cursors); got < 4 {
		t.Errorf("read environment pages %d times, want a re-read after the deployment event", got)
	}
	out := view(model)
	for _, want := range []string{"just now deployment completed Billing / Development · provision · ana", "30m instance created Care / Development"} {
		if !strings.Contains(out, want) {
			t.Errorf("ticker is missing %q:\n%s", want, out)
		}
	}
	ticker := ""
	for _, line := range strings.Split(ansi.Strip(model.View()), "\n") {
		if strings.Contains(line, "LIVE") {
			ticker = line
		}
	}
	if strings.Count(ticker, "instance created") != 1 || strings.Count(ticker, "deployment completed") != 1 {
		t.Errorf("an event read twice should show once:\n%s", ticker)
	}
	if !strings.Contains(out, "instance updated") || !strings.Contains(out, "more than 2 events since the last check") {
		t.Errorf("a cut-short poll should show its events and say it was cut short:\n%s", out)
	}
}

func TestModelMovesSelection(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	seen := map[string]bool{m.Selected(): true}
	for _, key := range []string{"tab", "tab", "tab", "tab", "tab", "tab", "up", "down", "left", "right"} {
		m = press(m, key)
		seen[m.Selected()] = true
	}
	if len(seen) < 6 {
		t.Errorf("visited %v, want every environment", seen)
	}
}

func TestModelQuitAndRefresh(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 200, 40)
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); cmd == nil || cmd() != tea.Quit() {
		t.Error("q should quit")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("r should re-read the organization")
	}
	if !strings.Contains(ansi.Strip(next.View()), "⟳ …") {
		t.Error("the header should show a read in flight")
	}
	next, _ = next.Update(cmd())
	if out := ansi.Strip(next.View()); !strings.Contains(out, "⟳ just now") {
		t.Errorf("after the re-read the header should say when it read:\n%s", out)
	}
}

func TestModelNarrow(t *testing.T) {
	out := view(loaded(t, fixture(), config(&opener{}), 70, 20))
	if strings.Contains(out, "INSTANCES") {
		t.Error("a narrow terminal should drop the side panel")
	}
	if !strings.Contains(out, "alarming") {
		t.Error("a narrow header should keep the alarm count")
	}
}

func TestModelError(t *testing.T) {
	m := loaded(t, &fakeAPI{err: errors.New("unauthorized")}, config(&opener{}), 100, 30)
	if out := view(m); !strings.Contains(out, "Could not read the dashboard: list environments: unauthorized") {
		t.Errorf("view = %s", out)
	}
}
