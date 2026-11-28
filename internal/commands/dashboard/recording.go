package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
)

// Recording is one incident as the dashboard saw it: the organization in
// view when it started, its stash, and the events in its blast radius.
type Recording struct {
	Org          string                  `json:"org"`
	Focus        string                  `json:"focus"`
	Start        time.Time               `json:"start"`
	End          time.Time               `json:"end"`
	Alarm        FiringAlarm             `json:"alarm"`
	Environments []Environment           `json:"environments"`
	Runtimes     []Runtime               `json:"runtimes"`
	Applications []Application           `json:"applications"`
	Annotations  map[string][]Annotation `json:"annotations,omitempty"`
	Stash        []StashEntry            `json:"stash"`
	Events       []Event                 `json:"events"`
}

func newRecording(org string, r incidentRecord, now time.Time) Recording {
	events := r.timeline()
	end := now
	if !r.open() {
		end = r.resolved
	}
	if last := events[len(events)-1].At; last.After(end) {
		end = last
	}
	rec := Recording{
		Org: org, Focus: r.snap.Focus, Start: events[0].At, End: end, Alarm: r.alarm,
		Environments: r.snap.Environments, Runtimes: r.snap.Runtimes, Applications: r.snap.Applications,
		Stash: r.stash, Events: events,
	}
	for _, a := range r.snap.Annotations {
		if rec.Annotations == nil {
			rec.Annotations = map[string][]Annotation{}
		}
		rec.Annotations[a.ProjectID] = append(rec.Annotations[a.ProjectID], a)
	}
	return rec
}

// LoadRecording reads a recording written by the dashboard.
func LoadRecording(r io.Reader) (*Recording, error) {
	var rec Recording
	if err := json.NewDecoder(r).Decode(&rec); err != nil {
		return nil, fmt.Errorf("read the recording: %w", err)
	}
	if rec.Focus == "" || len(rec.Environments) == 0 {
		return nil, errors.New("read the recording: it holds no environments")
	}
	return &rec, nil
}

// Window is how far back a replay reaches to cover the whole recording.
func (rec *Recording) Window() time.Duration {
	span := rec.End.Sub(rec.Start)
	for _, w := range ReplayWindows {
		if w >= span {
			return w
		}
	}
	return span.Truncate(time.Hour) + time.Hour
}

// API serves the dashboard's reads from the recording.
func (rec *Recording) API() API { return recordedAPI{rec: rec} }

// PlaybackConfig sets cfg up to replay the recording, from the moment it
// started, with nothing read live.
func (rec *Recording) PlaybackConfig(cfg Config) Config {
	end := rec.End
	cfg.Org, cfg.Focus, cfg.Depth = rec.Org, rec.Focus, len(rec.Environments)+1
	cfg.Refresh, cfg.EventPoll, cfg.StashPoll = 0, 0, 0
	cfg.Watch, cfg.Links = false, nil
	cfg.Now = func() time.Time { return end }
	cfg.Replay = rec.Window()
	return cfg
}

type recordedAPI struct{ rec *Recording }

func (a recordedAPI) EnvironmentsByID(_ context.Context, ids []string, _ string) ([]Environment, string, error) {
	var out []Environment
	for _, env := range a.rec.Environments {
		if slices.Contains(ids, env.ID) {
			out = append(out, env)
		}
	}
	return out, "", nil
}

func (a recordedAPI) Directory(context.Context, string) ([]Place, string, error) {
	out := make([]Place, 0, len(a.rec.Environments))
	for _, env := range a.rec.Environments {
		out = append(out, Place{ID: env.ID, Name: env.Name, Project: env.Project})
	}
	return out, "", nil
}

func (a recordedAPI) RecentEnvironment(context.Context) (string, error) { return a.rec.Focus, nil }

func (a recordedAPI) Runtimes(context.Context, string) ([]Runtime, string, error) {
	return a.rec.Runtimes, "", nil
}

func (a recordedAPI) Applications(context.Context, string) ([]Application, string, error) {
	return a.rec.Applications, "", nil
}

func (a recordedAPI) StashEntries(_ context.Context, scopes []Scope, _ string) ([]StashEntry, string, error) {
	var out []StashEntry
	for _, e := range a.rec.Stash {
		if slices.Contains(scopes, e.Scope) {
			out = append(out, e)
		}
	}
	return out, "", nil
}

// SubscribeStash refuses as a live stream without a personal access token
// would, so the dashboard reads the recorded stash once and stops there.
func (a recordedAPI) SubscribeStash(context.Context, string, string) (*streaming.Stream[StashEvent], error) {
	return nil, streaming.ErrRequiresPAT
}

// A recording keeps no deployment logs.
func (a recordedAPI) DeploymentLogs(context.Context, string) (DeploymentLog, error) {
	return DeploymentLog{}, errNotRecorded
}

func (a recordedAPI) StreamDeploymentLogs(context.Context, string) (*streaming.Stream[LogBatch], error) {
	return nil, errNotRecorded
}

func (a recordedAPI) ApproveDeployment(context.Context, string) error { return ErrNoDeployer }

func (a recordedAPI) Deploy(context.Context, string) error { return ErrNoDeployer }

var errNotRecorded = errors.New("a recording has no deployment logs")

func (a recordedAPI) Annotations(_ context.Context, projectID, _ string) ([]Annotation, string, error) {
	return a.rec.Annotations[projectID], "", nil
}

func (a recordedAPI) Events(context.Context, time.Time) (Log, error) { return Log{}, nil }

func (a recordedAPI) History(_ context.Context, since time.Time) (Log, error) {
	var log Log
	for _, ev := range a.rec.Events {
		if !ev.At.Before(since) {
			log.Events = append(log.Events, ev)
		}
	}
	return log, nil
}
