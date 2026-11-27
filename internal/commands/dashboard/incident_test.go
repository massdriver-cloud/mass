package dashboard_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

type notes struct{ sent []string }

func (n *notes) notify(title, body string) { n.sent = append(n.sent, title+" | "+body) }

func watching(t *testing.T, api *fakeAPI, n *notes) dashboard.Model {
	t.Helper()
	cfg := config(&opener{})
	cfg.Watch, cfg.Notify = true, n.notify
	return loaded(t, api, cfg, 200, 48)
}

// reread re-reads the organization, as a refresh would, and runs what the
// read asks for.
func reread(t *testing.T, m dashboard.Model) dashboard.Model {
	t.Helper()
	next, cmd := pressCmd(m, "r")
	var model tea.Model = next
	model, cmd = model.Update(cmd())
	if cmd != nil {
		if batch, ok := cmd().(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					c()
				}
			}
		}
	}
	return model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
}

func setFiring(api *fakeAPI, firing bool) {
	queue := &api.envPages[0][1].Instances[1]
	queue.Alarms.Items = nil
	if firing {
		queue.Alarms.Items = []dashboard.Alarm{{DisplayName: "Queue backed up", CurrentState: &dashboard.AlarmState{Status: "ALARM", Message: "deep"}}}
	}
}

func TestIncident(t *testing.T) {
	api, n := fixture(), &notes{}
	m := press(watching(t, api, n), "2")
	if len(n.sent) != 0 {
		t.Fatalf("alarms already firing at start should not notify: %v", n.sent)
	}

	setFiring(api, true)
	m = reread(t, m)
	if m.ActiveTab() != dashboard.TabConstellation || m.Selected() != "care-dev" {
		t.Errorf("tab %d selected %s; an incident should take the view to its environment", m.ActiveTab(), m.Selected())
	}
	if len(n.sent) != 1 || n.sent[0] != "▲ Queue backed up | care-dev-queue · Care / Development" {
		t.Errorf("notified %v", n.sent)
	}
	out := view(m)
	for _, want := range []string{"▲ Queue backed up · care-dev-queue · Care / Development", "downstream · 0:00"} {
		if !strings.Contains(out, want) {
			t.Errorf("banner is missing %q:\n%s", want, out)
		}
	}
	if acked := view(press(m, "esc")); strings.Contains(acked, "Queue backed up ·") {
		t.Error("esc should put the banner away")
	}

	setFiring(api, false)
	m = reread(t, m)
	if out := view(m); !strings.Contains(out, "✓ Queue backed up on care-dev-queue resolved after 0:00") {
		t.Errorf("a cleared alarm should show as resolved:\n%s", out)
	}
	tick := frameTick(t)
	var model tea.Model = m
	for range 70 {
		model, _ = model.Update(tick)
	}
	back := model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
	if back.ActiveTab() != dashboard.TabRuntimes || strings.Contains(view(back), "resolved after") {
		t.Errorf("tab %d; once resolved, the view should go back where it was", back.ActiveTab())
	}
}

func TestIncidentKeepsWhereYouMoved(t *testing.T) {
	api := fixture()
	m := press(watching(t, api, &notes{}), "2")
	setFiring(api, true)
	m = press(reread(t, m), "3")
	setFiring(api, false)
	m = reread(t, m)
	tick := frameTick(t)
	var model tea.Model = m
	for range 70 {
		model, _ = model.Update(tick)
	}
	if got := model.(dashboard.Model).ActiveTab(); got != dashboard.TabAlarms { //nolint:errcheck,forcetypeassert // Update always returns a Model
		t.Errorf("tab = %d; a view someone moved during the incident should stay put", got)
	}
}

func settle(t *testing.T, m dashboard.Model) dashboard.Model {
	t.Helper()
	tick := frameTick(t)
	var model tea.Model = m
	for range 70 {
		model, _ = model.Update(tick)
	}
	return model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
}

func TestIncidentKeepsAMoveWhileResolved(t *testing.T) {
	api := fixture()
	m := press(watching(t, api, &notes{}), "2")
	setFiring(api, true)
	m = reread(t, m)
	setFiring(api, false)
	m = press(reread(t, m), "3")
	if got := settle(t, m).ActiveTab(); got != dashboard.TabAlarms {
		t.Errorf("tab = %d; a move during the resolved banner should stay put", got)
	}
}

func TestIncidentDuringResolvedKeepsTheFirstView(t *testing.T) {
	api := fixture()
	m := press(watching(t, api, &notes{}), "2")
	setFiring(api, true)
	m = reread(t, m)
	setFiring(api, false)
	m = reread(t, m)
	setFiring(api, true)
	m = reread(t, m)
	setFiring(api, false)
	m = reread(t, m)
	if got := settle(t, m).ActiveTab(); got != dashboard.TabRuntimes {
		t.Errorf("tab = %d; the view should go back to where it was before the first incident", got)
	}
}

func TestIncidentPastTheFirstPage(t *testing.T) {
	api, n := fixture(), &notes{}
	m := watching(t, api, n)
	queue := &api.envPages[0][1].Instances[1]
	queue.FiringAlarmCount = 4
	if out := view(reread(t, m)); !strings.Contains(out, "▲ 4 alarms firing · care-dev-queue") {
		t.Errorf("banner should name the instance's alarms:\n%s", out)
	}
	if len(n.sent) != 1 || !strings.HasPrefix(n.sent[0], "▲ 4 alarms firing | care-dev-queue") {
		t.Fatalf("notified %v; a rising count with nothing new listed should still open an incident", n.sent)
	}
}

func TestIncidentLeavingThePageStaysOpen(t *testing.T) {
	api, n := fixture(), &notes{}
	m := watching(t, api, n)
	setFiring(api, true)
	queue := &api.envPages[0][1].Instances[1]
	queue.FiringAlarmCount = 2
	m = reread(t, m)
	queue.Alarms.Items = nil
	m = reread(t, m)
	if len(n.sent) != 1 {
		t.Errorf("notified %v; the same count with a different page should open nothing", n.sent)
	}
	if out := view(m); strings.Contains(out, "resolved after") || !strings.Contains(out, "Queue backed up ·") {
		t.Errorf("an alarm that left the first page while the count held is not resolved:\n%s", out)
	}
	queue.FiringAlarmCount = 0
	m = reread(t, m)
	if out := view(m); !strings.Contains(out, "resolved after") {
		t.Errorf("a count of zero resolves it:\n%s", out)
	}
}

func TestIncidentNeedsWatch(t *testing.T) {
	api, n := fixture(), &notes{}
	cfg := config(&opener{})
	cfg.Notify = n.notify
	m := press(loaded(t, api, cfg, 200, 48), "2")
	setFiring(api, true)
	m = reread(t, m)
	if m.ActiveTab() != dashboard.TabRuntimes || len(n.sent) != 0 {
		t.Error("without --watch a new alarm should not move the view or notify")
	}
}

func TestIncidentClock(t *testing.T) {
	api, n := fixture(), &notes{}
	now := fixedNow()
	cfg := config(&opener{})
	cfg.Watch, cfg.Notify, cfg.Now = true, n.notify, func() time.Time { return now }
	m := loaded(t, api, cfg, 200, 48)
	setFiring(api, true)
	m = reread(t, m)
	now = now.Add(time.Hour + 2*time.Minute + 3*time.Second)
	if out := view(m); !strings.Contains(out, "1:02:03") {
		t.Errorf("banner should count up:\n%s", out)
	}
}
