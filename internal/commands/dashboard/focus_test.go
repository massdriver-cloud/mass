package dashboard_test

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

// drive runs cmd and what its messages ask for, for a number of steps.
func drive(t *testing.T, m dashboard.Model, cmd tea.Cmd, steps int) dashboard.Model {
	t.Helper()
	return pump(t, m, []tea.Cmd{cmd}, steps)
}

// keys presses keys and returns every command they asked for, batched.
func keys(m dashboard.Model, names ...string) (dashboard.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for _, k := range names {
		var cmd tea.Cmd
		m, cmd = pressCmd(m, k)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// tabTo steps the selection to id with tab, returning the command the last
// step asked for.
func tabTo(t *testing.T, m dashboard.Model, id string) (dashboard.Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for range 8 {
		if m.Selected() == id {
			return m, cmd
		}
		m, cmd = pressCmd(m, "tab")
	}
	t.Fatalf("tab never reached %s", id)
	return m, nil
}

func TestRefocusOnSelection(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	if m.Focus() != "care-dev" {
		t.Fatalf("focus = %q", m.Focus())
	}
	m, cmd := tabTo(t, m, "billing-dev")
	if m.Focus() != "care-dev" {
		t.Fatalf("focus %q; a selection should not move the focus at once", m.Focus())
	}
	m = drive(t, m, cmd, 30)
	if m.Focus() != "billing-dev" {
		t.Fatalf("focus = %q; a selection that rests should become the focus", m.Focus())
	}
	if out := view(m); !strings.Contains(out, "not loaded") {
		t.Errorf("environments past the new edge should show as not loaded:\n%s", out)
	}
}

func TestRefocusSkipsASelectionThatMoved(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	m, first := tabTo(t, m, "billing-dev")
	m = press(m, "tab")
	m = drive(t, m, first, 30)
	if m.Focus() != "care-dev" {
		t.Errorf("focus = %q; a selection that moved on should not take the focus", m.Focus())
	}
}

func TestSearchFocusesOnEnter(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	m, typing := keys(m, "/", "b", "i", "l")
	m = drive(t, m, typing, 30)
	if m.Focus() != "care-dev" {
		t.Fatalf("focus = %q; a pause while typing should not move the focus", m.Focus())
	}
	m, cmd := keys(m, "l", "i", "n", "g", "-", "d", "enter")
	if m.Focus() != "billing-dev" {
		t.Fatalf("focus = %q; enter on a search should move the focus at once", m.Focus())
	}
	m = drive(t, m, cmd, 30)
	if m.Focus() != "billing-dev" || m.Selected() != "billing-dev" {
		t.Errorf("focus %q selected %q", m.Focus(), m.Selected())
	}
}

func TestOlderFailedReadKeepsTheNewFocus(t *testing.T) {
	api := fixture()
	m := loaded(t, api, config(&opener{}), 160, 48)
	m, refresh := pressCmd(m, "r")
	m, cmd := keys(m, "/", "p", "l", "a", "t", "enter")
	api.err = errors.New("timeout")
	m = drive(t, m, refresh, 1)
	api.err = nil
	m = drive(t, m, cmd, 30)
	if m.Focus() != "platform-dev" {
		t.Errorf("focus = %q; a failed read for the old focus should not undo the new one", m.Focus())
	}
}

func TestEnterFocusesAtOnce(t *testing.T) {
	m := press(loaded(t, fixture(), config(&opener{}), 160, 48), "/", "p", "l", "a", "t", "enter")
	m, cmd := pressCmd(m, "enter")
	m = drive(t, m, cmd, 3)
	if m.Focus() != "platform-dev" {
		t.Errorf("focus = %q; enter should move the focus to the selection", m.Focus())
	}
}

func TestSearchBeyondTheView(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 160, 48)
	m, cmd := keys(m, "/", "d", "e", "m", "o", "enter")
	m = drive(t, m, cmd, 30)
	if m.Focus() != "sandbox-demo" {
		t.Fatalf("focus = %q; a search for an environment out of view should move there", m.Focus())
	}
	m, cmd = keys(m, "/", "z", "z", "z", "enter")
	m = drive(t, m, cmd, 30)
	if out := view(m); !strings.Contains(out, "no environment matches zzz") {
		t.Errorf("a search with no match should say so:\n%s", out)
	}
}

func TestFocusMovesNodes(t *testing.T) {
	m := press(loaded(t, fixture(), config(&opener{}), 160, 48), "/", "p", "l", "a", "t", "enter")
	m, cmd := pressCmd(m, "enter")
	m = drive(t, m, cmd, 3)
	if m.Focus() != "platform-dev" {
		t.Fatalf("focus = %q", m.Focus())
	}
	start := view(m)
	tick := frameTick(t)
	var model tea.Model = m
	for range 20 {
		model, _ = model.Update(tick)
	}
	if end := view(model.(dashboard.Model)); end == start { //nolint:errcheck,forcetypeassert // Update always returns a Model
		t.Error("nodes should glide to their new places after the focus moves")
	}
}
