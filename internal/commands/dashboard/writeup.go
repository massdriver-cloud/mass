package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type writtenMsg struct {
	path string
	err  error
}

// WriteFile writes a file readable only by its owner, since an incident's
// stash values and alarm messages can be sensitive.
func WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o600)
}

func (m Model) writeIncident() (Model, tea.Cmd) {
	r, ok := m.incidentToWrite()
	if !ok {
		m.toast, m.toastAt = "no incident to write up", m.frame
		return m, nil
	}
	write := m.cfg.WriteFile
	if write == nil {
		return m, nil
	}
	now := m.cfg.Now()
	name := incidentName(r)
	md := writeup(r, name, now)
	rec, err := json.MarshalIndent(newRecording(m.cfg.Org, r, now), "", "  ")
	if err != nil {
		m.toast, m.toastAt = "could not record the incident: "+err.Error(), m.frame
		return m, nil
	}
	return m, func() tea.Msg {
		if err := write(name+".json", rec); err != nil {
			return writtenMsg{path: name + ".json", err: err}
		}
		return writtenMsg{path: name + ".md", err: write(name+".md", []byte(md))}
	}
}

func (m Model) applyWritten(msg writtenMsg) Model {
	m.toast, m.toastAt = "✎ wrote "+msg.path, m.frame
	if msg.err != nil {
		m.toast = "could not write " + msg.path + ": " + clean(msg.err.Error())
	}
	return m
}

// An alarm firing since before the dashboard opened, or without --watch, has
// no record yet, so one is made from what is in view.
func (m Model) incidentToWrite() (incidentRecord, bool) {
	if m.graph == nil {
		return incidentRecord{}, false
	}
	var alarm *FiringAlarm
	switch m.tab {
	case TabAlarms:
		if len(m.graph.Firing) > 0 {
			alarm = &m.graph.Firing[m.alarmIdx]
		}
	case TabConstellation:
		if env := m.graph.Env(m.selected); env != nil && len(env.Firing) > 0 {
			alarm = &env.Firing[0]
		}
	case TabRuntimes, TabUpgrades:
	}
	match := func(r incidentRecord) bool {
		switch {
		case alarm != nil:
			return alarmKey(r.alarm) == alarmKey(*alarm)
		case m.tab == TabConstellation:
			return r.alarm.EnvID == m.selected
		}
		return false
	}
	for _, r := range slices.Backward(m.incidents) {
		if match(r) {
			return r, true
		}
	}
	if alarm != nil {
		started := alarm.Since
		if started.IsZero() {
			started = m.cfg.Now()
		}
		return m.newRecord(*alarm, started, started), true
	}
	if len(m.incidents) > 0 {
		return m.incidents[len(m.incidents)-1], true
	}
	return incidentRecord{}, false
}

func incidentName(r incidentRecord) string {
	return "incident-" + fileSafe(r.alarm.EnvID) + "-" + r.started.UTC().Format("20060102-1504")
}

func fileSafe(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		}
		return '-'
	}, s)
}

func writeup(r incidentRecord, name string, now time.Time) string {
	g := BuildGraph(r.snap)
	a := r.alarm
	var b strings.Builder
	fmt.Fprintf(&b, "# %s on %s\n\n", a.Name, a.InstanceID)
	b.WriteString("| | |\n|---|---|\n")
	fmt.Fprintf(&b, "| Environment | %s (`%s`) |\n", mdCell(envLabel(g, a.EnvID)), a.EnvID)
	fmt.Fprintf(&b, "| Instance | `%s` |\n", a.InstanceID)
	fmt.Fprintf(&b, "| Started | %s |\n", r.started.UTC().Format("2006-01-02 15:04:05 UTC"))
	if r.open() {
		fmt.Fprintf(&b, "| Resolved | still firing at %s |\n", now.UTC().Format("2006-01-02 15:04:05 UTC"))
		fmt.Fprintf(&b, "| Duration | %s so far |\n", clock(now.Sub(r.started)))
	} else {
		fmt.Fprintf(&b, "| Resolved | %s |\n", r.resolved.UTC().Format("2006-01-02 15:04:05 UTC"))
		fmt.Fprintf(&b, "| Duration | %s |\n", clock(r.resolved.Sub(r.started)))
	}
	b.WriteString("\n## Alarm\n\n**" + mdCell(a.Name) + "**\n")
	if a.Message != "" {
		b.WriteString("\n> " + strings.Join(strings.Fields(a.Message), " ") + "\n")
	}
	writeBlastRadius(&b, g, r)
	writeTimeline(&b, g, r)
	b.WriteString("\n## Root cause\n\nTBD\n\n## Follow-ups\n\n- [ ] TBD\n")
	b.WriteString("\n## Playback\n\n```shell\nmass dashboard --playback " + name + ".json\n```\n")
	return b.String()
}

func writeBlastRadius(b *strings.Builder, g *Graph, r incidentRecord) {
	envs, insts := r.blast.Downstream()
	b.WriteString("\n## Blast radius\n\n")
	if insts == 0 {
		b.WriteString("Nothing in view depends on `" + r.alarm.InstanceID + "`.\n")
		return
	}
	fmt.Fprintf(b, "%s and %s downstream of `%s`:\n\n", count(envs, "environment"), count(insts, "instance"), r.alarm.InstanceID)
	byEnv := map[string][]string{}
	for id, hop := range r.blast.Instances {
		if hop == 0 {
			continue
		}
		env := ""
		if inst := g.Instance(id); inst != nil {
			env = inst.EnvID
		}
		byEnv[env] = append(byEnv[env], fmt.Sprintf("`%s` (%s)", id, count(hop, "hop")))
	}
	ids := make([]string, 0, len(byEnv))
	for id := range byEnv {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(x, y string) int {
		if hx, hy := r.blast.Envs[x], r.blast.Envs[y]; hx != hy {
			return hx - hy
		}
		return strings.Compare(x, y)
	})
	for _, id := range ids {
		slices.Sort(byEnv[id])
		fmt.Fprintf(b, "- %s: %s\n", mdCell(envLabel(g, id)), strings.Join(byEnv[id], ", "))
	}
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func writeTimeline(b *strings.Builder, g *Graph, r incidentRecord) {
	b.WriteString("\n## Timeline\n\n| Time (UTC) | Event | Where | Detail | By |\n|---|---|---|---|---|\n")
	day := r.started.UTC().Format(time.DateOnly)
	for _, ev := range r.timeline() {
		at := ev.At.UTC().Format(time.TimeOnly)
		if ev.At.UTC().Format(time.DateOnly) != day {
			at = ev.At.UTC().Format(time.DateTime)
		}
		where := ""
		if ev.EnvID != "" {
			where = envLabel(g, ev.EnvID)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", at, EventLabel(ev), mdCell(where), mdCell(ev.Detail), mdCell(ev.Actor))
	}
	if r.dropped > 0 {
		fmt.Fprintf(b, "\nNot listed: %s later in the blast radius.\n", count(r.dropped, "event"))
	}
}

func mdCell(s string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "|", `\|`)
}
