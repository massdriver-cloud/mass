package dashboard

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	lockOnFrames   = 16
	resolvedFrames = 60
)

// Incident is an alarm that started firing while the dashboard watched.
type Incident struct {
	Alarm   FiringAlarm
	Started time.Time
	// Unlisted marks an incident whose alarm is past the first page of its
	// instance's alarms; it stands for the instance's alarms as a whole.
	Unlisted bool
}

type watchState struct {
	open     []Incident
	resolved *resolvedIncident
	lockedAt int
	acked    bool
	// touched records a keypress during an incident, so the dashboard does
	// not pull the view back from where someone moved it.
	touched bool
	back    *viewState
}

type resolvedIncident struct {
	incident Incident
	after    time.Duration
	at       int
}

type viewState struct {
	tab      Tab
	selected string
	blastOn  bool
}

func alarmKey(a FiringAlarm) string { return a.InstanceID + "\x00" + a.Name }

// watchAlarms compares the alarms firing before a read with the ones firing
// after it. The alarm list is one page per instance, so an instance's count
// decides: an incident opens only when the count rises, and an alarm missing
// from the list resolves only when the list holds every alarm of its instance.
func (m Model) watchAlarms(before *Graph) (Model, tea.Cmd) {
	if !m.cfg.Watch || before == nil {
		return m, nil
	}
	listed := map[string]bool{}
	for _, a := range m.graph.Firing {
		listed[alarmKey(a)] = true
	}
	m = m.resolveIncidents(listed)
	started := startedIncidents(before, m.graph, m.cfg.Now())
	if len(started) == 0 {
		return m, nil
	}
	return m.openIncidents(started)
}

func startedIncidents(before, after *Graph, now time.Time) []Incident {
	was := map[string]bool{}
	for _, a := range before.Firing {
		was[alarmKey(a)] = true
	}
	byInstance := map[string][]FiringAlarm{}
	for _, a := range after.Firing {
		byInstance[a.InstanceID] = append(byInstance[a.InstanceID], a)
	}
	var started []Incident
	for _, env := range after.Envs {
		for _, inst := range env.Instances {
			prev := 0
			if old := before.Instance(inst.ID); old != nil {
				prev = old.Firing
			}
			if inst.Firing <= prev {
				continue
			}
			fresh := newAlarms(byInstance[inst.ID], was, now)
			started = append(started, fresh...)
			if len(fresh) == 0 {
				started = append(started, Incident{Started: now, Unlisted: true, Alarm: FiringAlarm{
					InstanceID: inst.ID, EnvID: env.ID, Name: fmt.Sprintf("%d alarms firing", inst.Firing),
				}})
			}
		}
	}
	return started
}

func newAlarms(alarms []FiringAlarm, was map[string]bool, now time.Time) []Incident {
	var out []Incident
	for _, a := range alarms {
		if !was[alarmKey(a)] {
			out = append(out, Incident{Alarm: a, Started: now})
		}
	}
	return out
}

func (m Model) resolveIncidents(listed map[string]bool) Model {
	w := m.watch
	open := w.open[:0:0]
	for _, inc := range w.open {
		if m.stillFiring(inc, listed) {
			open = append(open, inc)
			continue
		}
		w.resolved = &resolvedIncident{incident: inc, after: m.cfg.Now().Sub(inc.Started), at: m.frame}
	}
	w.open = open
	m.watch = w
	return m
}

func (m Model) stillFiring(inc Incident, listed map[string]bool) bool {
	inst := m.graph.Instance(inc.Alarm.InstanceID)
	if inst == nil || inst.Firing == 0 {
		return false
	}
	if inc.Unlisted || listed[alarmKey(inc.Alarm)] {
		return true
	}
	shown := 0
	for _, a := range m.graph.Firing {
		if a.InstanceID == inst.ID {
			shown++
		}
	}
	return inst.Firing > shown
}

func (m Model) openIncidents(started []Incident) (Model, tea.Cmd) {
	w := m.watch
	if w.back == nil {
		w.back = &viewState{tab: m.tab, selected: m.selected, blastOn: m.blastOn}
		w.touched = false
	}
	w.open = slices.Clone(w.open)
	started = foldUnlisted(started, w.open)
	if len(started) == 0 {
		m.watch = w
		return m, nil
	}
	w.open = append(started, w.open...)
	w.resolved, w.acked, w.lockedAt = nil, false, m.frame
	m.watch = w
	newest := started[0].Alarm
	m.tab, m.selected, m.blastOn = TabConstellation, newest.EnvID, true
	return m, m.notify("▲ "+newest.Name, newest.InstanceID+" · "+envLabel(m.graph, newest.EnvID))
}

// foldUnlisted renames an instance's open unlisted incident to its new count,
// in open, rather than opening a second one for the same instance.
func foldUnlisted(started, open []Incident) []Incident {
	out := started[:0:0]
	for _, inc := range started {
		folded := false
		for i := range open {
			if inc.Unlisted && open[i].Unlisted && open[i].Alarm.InstanceID == inc.Alarm.InstanceID {
				open[i].Alarm.Name, folded = inc.Alarm.Name, true
			}
		}
		if !folded {
			out = append(out, inc)
		}
	}
	return out
}

func (m Model) notify(title, body string) tea.Cmd {
	send := m.cfg.Notify
	if send == nil {
		return nil
	}
	return func() tea.Msg {
		send(title, body)
		return nil
	}
}

// settleWatch puts the view back where it was once the last incident has
// been resolved for a moment, unless someone moved it meanwhile.
func (m Model) settleWatch() Model {
	w := m.watch
	if w.resolved == nil || len(w.open) > 0 || m.frame-w.resolved.at < resolvedFrames {
		return m
	}
	if w.back != nil && !w.touched {
		m.tab, m.selected, m.blastOn = w.back.tab, w.back.selected, w.back.blastOn
	}
	m.watch = watchState{}
	return m
}

func (m Model) lockOn() (string, int) {
	w := m.watch
	if len(w.open) == 0 || m.frame-w.lockedAt >= lockOnFrames {
		return "", 0
	}
	return w.open[0].Alarm.EnvID, m.frame - w.lockedAt
}

func (m Model) incidentBanner() string {
	w := m.watch
	switch {
	case len(w.open) > 0 && !w.acked:
		inc := w.open[0]
		envs, insts := m.graph.BlastRadius(inc.Alarm.InstanceID).Downstream()
		text := fmt.Sprintf(" ▲ %s · %s · %s · %d environments, %d instances downstream · %s ",
			inc.Alarm.Name, inc.Alarm.InstanceID, envLabel(m.graph, inc.Alarm.EnvID), envs, insts,
			clock(m.cfg.Now().Sub(inc.Started)))
		if more := len(w.open) - 1; more > 0 {
			text += fmt.Sprintf("· +%d more ", more)
		}
		bg := colorRed
		if m.frame%12 >= 6 {
			bg = colorRedDim
		}
		return lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Background(bg).Width(m.width).Render(text)
	case w.resolved != nil:
		r := w.resolved
		text := fmt.Sprintf(" ✓ %s on %s resolved after %s ", r.incident.Alarm.Name, r.incident.Alarm.InstanceID, clock(r.after))
		return lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(colorGreen).Width(m.width).Render(text)
	}
	return ""
}

// clock formats a duration as m:ss, or h:mm:ss past an hour.
func clock(d time.Duration) string {
	d = d.Round(time.Second)
	h, mins, secs := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, mins, secs)
	}
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func overlayFirstLine(body, line string) string {
	if line == "" {
		return body
	}
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return line + body[i:]
	}
	return line
}

// drawLockOn closes rings and crosshairs in on the environment an incident
// just took the view to.
func (s Scene) drawLockOn(c *Canvas) {
	center, ok := s.Pos[s.Target]
	if s.Target == "" || !ok {
		return
	}
	t := float64(s.TargetAge) / lockOnFrames
	r := nodeRadius + 4 + (1-t)*40
	color := colorRed
	if s.TargetAge%2 == 1 {
		color = colorHighlight
	}
	c.Ring(center, r, func(f float64) lipgloss.Color {
		if int(f*16)%2 == 0 {
			return color
		}
		return ""
	}, layerHalo)
	for _, d := range []Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		c.Line(center.X+d.X*(r+1), center.Y+d.Y*(r+1), center.X+d.X*(r+5), center.Y+d.Y*(r+5), color, layerHalo, 0)
	}
}
