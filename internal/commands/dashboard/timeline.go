package dashboard

import (
	"slices"
	"strings"
	"time"
)

const (
	incidentsKept = 20
	timelineKept  = 200
	// incidentLead reaches back before an incident for what led up to it,
	// such as the deployment that set it off.
	incidentLead = 15 * time.Minute
)

// incidentRecord is what the dashboard saw of one incident: the view when it
// started, and what happened in its blast radius until it cleared.
type incidentRecord struct {
	alarm    FiringAlarm
	started  time.Time
	resolved time.Time
	snap     *Snapshot
	stash    []StashEntry
	blast    *Blast
	events   []Event
	// dropped counts events past timelineKept. The earliest are kept, since
	// they are nearest the cause.
	dropped int
}

func (r incidentRecord) open() bool { return r.resolved.IsZero() }

func (r incidentRecord) is(inc Incident) bool {
	return alarmKey(r.alarm) == alarmKey(inc.Alarm) && r.started.Equal(inc.Started)
}

func (r incidentRecord) add(ev Event) incidentRecord {
	if _, ok := r.blast.Envs[ev.EnvID]; !ok {
		return r
	}
	if len(r.events) >= timelineKept {
		r.dropped++
		return r
	}
	r.events = append(slices.Clip(r.events), ev)
	return r
}

func (r incidentRecord) timeline() []Event {
	id := "incident:" + r.alarm.InstanceID + ":" + r.started.UTC().Format(time.RFC3339)
	out := make([]Event, 0, len(r.events)+2)
	out = append(out, Event{ID: id + ":fired", At: r.started, Type: "alarm.fired", EnvID: r.alarm.EnvID, Detail: r.alarm.Name})
	out = append(out, r.events...)
	if !r.open() {
		out = append(out, Event{ID: id + ":cleared", At: r.resolved, Type: "alarm.cleared", EnvID: r.alarm.EnvID, Detail: r.alarm.Name})
	}
	slices.SortStableFunc(out, func(a, b Event) int { return a.At.Compare(b.At) })
	return out
}

func (m Model) newRecord(a FiringAlarm, started, from time.Time) incidentRecord {
	r := incidentRecord{alarm: a, started: started, snap: m.snap, stash: m.stash.entries(), blast: m.graph.BlastRadius(a.InstanceID)}
	for _, ev := range slices.Backward(m.events) {
		if !ev.At.Before(from) {
			r = r.add(ev)
		}
	}
	return r
}

func (m Model) recordIncidents(started []Incident) Model {
	records := slices.Clone(m.incidents)
	for _, inc := range started {
		records = append(records, m.newRecord(inc.Alarm, inc.Started, inc.Started.Add(-incidentLead)))
	}
	m.incidents = records[max(len(records)-incidentsKept, 0):]
	return m
}

func (m Model) recordResolved(inc Incident, at time.Time) Model {
	records := slices.Clone(m.incidents)
	for i, r := range records {
		if r.open() && r.is(inc) {
			records[i].resolved = at
		}
	}
	m.incidents = records
	return m
}

func (m Model) recordEvents(fresh []Event) Model {
	if len(fresh) == 0 || !slices.ContainsFunc(m.incidents, incidentRecord.open) {
		return m
	}
	oldest := slices.SortedStableFunc(slices.Values(fresh), func(a, b Event) int { return a.At.Compare(b.At) })
	records := slices.Clone(m.incidents)
	for i, r := range records {
		if !r.open() {
			continue
		}
		for _, ev := range oldest {
			r = r.add(ev)
		}
		records[i] = r
	}
	m.incidents = records
	return m
}

func (st Stash) entries() []StashEntry {
	var out []StashEntry
	for _, keys := range st {
		for _, e := range keys {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b StashEntry) int {
		return strings.Compare(a.Scope.String()+"\x00"+a.Key, b.Scope.String()+"\x00"+b.Key)
	})
	return out
}
