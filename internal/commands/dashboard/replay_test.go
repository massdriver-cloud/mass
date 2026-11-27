package dashboard_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

// frameTick is a message from the animation clock, taken from a model's
// first commands.
func frameTick(t *testing.T) tea.Msg {
	t.Helper()
	batch, ok := dashboard.NewModel(fixture(), dashboard.Config{Now: fixedNow}).Init()().(tea.BatchMsg)
	if !ok {
		t.Fatal("Init should batch its commands")
	}
	for _, cmd := range batch {
		if msg := cmd(); fmt.Sprintf("%T", msg) == "dashboard.frameMsg" {
			return msg
		}
	}
	t.Fatal("Init should start the animation clock")
	return nil
}

// replayed picks a window with pick keys, plays it, presses keys, and runs
// the animation clock for some frames.
func replayed(t *testing.T, api *fakeAPI, pick []string, frames int, keys ...string) dashboard.Model {
	t.Helper()
	m, cmd := pressCmd(loaded(t, api, config(&opener{}), 200, 48), append([]string{"t"}, pick...)...)
	if cmd == nil {
		t.Fatal("picking a window should read the audit log")
	}
	next, _ := m.Update(cmd())
	m = press(next.(dashboard.Model), keys...) //nolint:errcheck,forcetypeassert // Update always returns a Model
	tick := frameTick(t)
	var model tea.Model = m
	for range frames {
		model, _ = model.Update(tick)
	}
	return model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
}

func withHistory(events []dashboard.Event, err error) *fakeAPI {
	api := fixture()
	api.history, api.historyErr = events, err
	return api
}

func TestReplay(t *testing.T) {
	at := fixedNow().Add(-3 * time.Hour)
	history := []dashboard.Event{
		{ID: "1", Type: "instance.created", EnvID: "care-dev", At: at},
		{ID: "2", Type: "deployment.started", EnvID: "billing-dev", At: at.Add(time.Minute)},
		{ID: "3", Type: "deployment.failed", EnvID: "billing-dev", At: at.Add(2 * time.Minute)},
	}
	enter := []string{"enter"}
	tests := []struct {
		name      string
		api       *fakeAPI
		pick      []string
		frames    int
		keys      []string
		want      []string
		wantSince time.Duration
	}{
		{name: "starts an hour back", api: withHistory(history, nil), pick: enter, want: []string{"REPLAY", "0/3 ×1", "t stop replay"}, wantSince: time.Hour},
		{name: "further back", api: withHistory(history, nil), pick: []string{"right", "l", "enter"}, wantSince: 3 * time.Hour},
		{name: "nearer, stopping at five minutes", api: withHistory(history, nil), pick: []string{"left", "left", "left", "left", "left", "h", "h", " "}, wantSince: 5 * time.Minute},
		{name: "plays", api: withHistory(history, nil), pick: enter, frames: 4, want: []string{"2/3 ×1", "deployment started Billing / Development"}},
		{name: "faster", api: withHistory(history, nil), pick: enter, frames: 2, keys: []string{"]"}, want: []string{"2/3 ×2"}},
		{name: "slower stays at one", api: withHistory(history, nil), pick: enter, keys: []string{"[", "["}, want: []string{"×1"}},
		{name: "paused", api: withHistory(history, nil), pick: enter, frames: 4, keys: []string{" "}, want: []string{"0/3 paused"}},
		{name: "done", api: withHistory(history, nil), pick: enter, frames: 10, want: []string{"3/3 done", "deployment failed"}},
		{name: "empty", api: withHistory(nil, nil), pick: enter, want: []string{"nothing happened in the last hour"}},
		{name: "failed", api: withHistory(nil, errors.New("forbidden")), pick: enter, want: []string{"could not read the audit log: forbidden"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := view(replayed(t, tc.api, tc.pick, tc.frames, tc.keys...))
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("view is missing %q:\n%s", want, out)
				}
			}
			if tc.wantSince > 0 {
				if got := fixedNow().Sub(tc.api.since); got != tc.wantSince {
					t.Errorf("read the audit log from %v back, want %v", got, tc.wantSince)
				}
			}
		})
	}
}

func TestReplayPicker(t *testing.T) {
	m := press(loaded(t, fixture(), config(&opener{}), 200, 48), "t")
	out := view(m)
	for _, want := range []string{"how far back?", "5m", "45m", "1h", "16h", "24h", "enter play", "esc cancel"} {
		if !strings.Contains(out, want) {
			t.Errorf("picker is missing %q:\n%s", want, out)
		}
	}
	if m.Selected() != "platform-dev" || press(m, "down").Selected() != "platform-dev" {
		t.Error("arrows pick a window, not an environment")
	}
	for _, cancel := range []string{"esc", "t"} {
		if out := view(press(m, cancel)); strings.Contains(out, "how far back?") || !strings.Contains(out, "LIVE") {
			t.Errorf("%s should close the picker:\n%s", cancel, out)
		}
	}
	if _, cmd := pressCmd(m, "q"); cmd == nil || cmd() != tea.Quit() {
		t.Error("q should still quit")
	}
}

func TestReplayStops(t *testing.T) {
	m := replayed(t, withHistory([]dashboard.Event{{ID: "1", Type: "instance.created"}}, nil), []string{"enter"}, 0)
	out := view(press(m, "t"))
	if strings.Contains(out, "REPLAY") || !strings.Contains(out, "LIVE") {
		t.Errorf("t should stop the replay and bring back the live ticker:\n%s", out)
	}
	if out := view(press(m, "t", "2", "t")); strings.Contains(out, "how far back?") {
		t.Error("replay belongs to the constellation tab")
	}
	failed := replayed(t, withHistory(nil, errors.New("forbidden")), []string{"enter"}, 0)
	if out := view(press(failed, "t")); strings.Contains(out, "could not read") {
		t.Error("t should clear a failed replay")
	}
}

func TestStartSpeed(t *testing.T) {
	tests := map[int]int{1: 1, 300: 1, 301: 2, 1000: 4, 100_000: 16}
	for n, want := range tests {
		m := replayed(t, withHistory(make([]dashboard.Event, n), nil), []string{"enter"}, 0)
		if out := view(m); !strings.Contains(out, fmt.Sprintf("0/%d ×%d", n, want)) {
			t.Errorf("%d events should start at ×%d:\n%s", n, want, out)
		}
	}
}

func TestWindowLabel(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Minute, "5m"}, {45 * time.Minute, "45m"}, {time.Hour, "1h"}, {16 * time.Hour, "16h"},
	}
	for _, tc := range tests {
		if got := dashboard.WindowLabel(tc.d); got != tc.want {
			t.Errorf("WindowLabel(%v) = %s, want %s", tc.d, got, tc.want)
		}
	}
}
