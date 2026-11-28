package dashboard_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func recorded(t *testing.T) *dashboard.Recording {
	t.Helper()
	d := &disk{}
	m := incidentScenario(t, d)
	writeUp(t, m)
	rec, err := dashboard.LoadRecording(strings.NewReader(d.files["incident-care-dev-app-latency-p99-20261126-0001.json"]))
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestRecording(t *testing.T) {
	rec := recorded(t)
	if rec.Org != "acme" || rec.Focus != "care-dev" || rec.Alarm.Name != "Latency | p99" {
		t.Errorf("org %q focus %q alarm %q", rec.Org, rec.Focus, rec.Alarm.Name)
	}
	if want := fixedNow().Add(-5 * time.Minute); !rec.Start.Equal(want) {
		t.Errorf("start %v, want the lead-up at %v", rec.Start, want)
	}
	if want := fixedNow().Add(6 * time.Minute); !rec.End.Equal(want) {
		t.Errorf("end %v, want when it cleared at %v", rec.End, want)
	}
	var types []string
	for _, ev := range rec.Events {
		types = append(types, ev.Type)
	}
	if got := strings.Join(types, " "); got != "deployment.completed alarm.fired deployment.failed alarm.cleared" {
		t.Errorf("events %s", got)
	}
	if len(rec.Stash) != 1 || rec.Stash[0].Key != "release" || *rec.Stash[0].Value != "v42" {
		t.Errorf("stash %+v", rec.Stash)
	}
	if len(rec.Environments) != 3 || len(rec.Runtimes) != 1 || len(rec.Applications) != 3 {
		t.Errorf("%d environments, %d runtimes, %d applications", len(rec.Environments), len(rec.Runtimes), len(rec.Applications))
	}
}

func TestRecordingAPI(t *testing.T) {
	api := recorded(t).API()
	ctx := context.Background()
	snap, err := dashboard.Fetch(ctx, api, dashboard.Where{})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Focus != "care-dev" || len(snap.Environments) != 3 {
		t.Errorf("focus %s, %d environments", snap.Focus, len(snap.Environments))
	}
	read := dashboard.ReadStash(ctx, api, snap.Environments)
	if len(read.Stash) != 1 || read.Err != nil {
		t.Errorf("stash %v err %v", read.Stash, read.Err)
	}
	log, _ := api.History(ctx, fixedNow())
	if len(log.Events) != 3 {
		t.Errorf("history since the alarm fired has %d events, want 3", len(log.Events))
	}
	if log, _ := api.Events(ctx, time.Time{}); len(log.Events) != 0 {
		t.Error("a recording has no live events")
	}
	if _, err := api.SubscribeStash(ctx, "care", "care-dev"); err == nil {
		t.Error("a recording has no live stash")
	}
}

func TestRecordingDeployments(t *testing.T) {
	rec := &dashboard.Recording{Focus: "e", Environments: []dashboard.Environment{{ID: "e", Instances: []dashboard.Instance{
		{ID: "i1", LatestDeployment: &dashboard.Deployment{ID: "d1", Status: "PROPOSED"}},
		{ID: "i2", LatestDeployment: &dashboard.Deployment{ID: "d2", Status: "COMPLETED"}},
		{ID: "i3"},
	}}}}
	api := rec.API()
	ctx := context.Background()
	got, _, err := api.Deployments(ctx, dashboard.DeploymentQuery{Instances: []string{"i1", "i2", "i3"}, Statuses: []string{"PROPOSED"}}, "")
	if err != nil || len(got) != 1 || got[0].ID != "d1" || got[0].Instance.ID != "i1" {
		t.Errorf("Deployments = %+v, %v", got, err)
	}
	if err := api.RejectDeployment(ctx, "d1", "no"); !errors.Is(err, dashboard.ErrNoDeployer) {
		t.Errorf("RejectDeployment = %v; a recording changes nothing", err)
	}
	if _, err := api.PeekDeployment(ctx, "i1", "d1"); err == nil {
		t.Error("a recording has no deployment configuration")
	}
}

func TestPlayback(t *testing.T) {
	rec := recorded(t)
	m := loaded(t, rec.API(), rec.PlaybackConfig(dashboard.Config{}), 200, 48)
	if out := view(m); !strings.Contains(out, "REPLAY") || !strings.Contains(out, "/4") || strings.Contains(out, "LIVE") {
		t.Errorf("playback should open on the replay:\n%s", out)
	}
	tick := frameTick(t)
	var model tea.Model = m
	for range 20 {
		model, _ = model.Update(tick)
	}
	m = model.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
	if out := view(m); !strings.Contains(out, "4/4 done") || !strings.Contains(out, "✅ Care / Development · Latency | p99") {
		t.Errorf("playback should run to the end:\n%s", out)
	}
	if out := view(press(m, "3")); !strings.Contains(out, "Latency | p99") {
		t.Errorf("the map should show the alarm firing:\n%s", out)
	}
	if out := view(press(m, "t")); strings.Contains(out, "REPLAY") {
		t.Errorf("t should stop the replay:\n%s", out)
	}
}

func TestLoadRecording(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "not json", in: "# incident", want: "read the recording"},
		{name: "empty", in: "{}", want: "holds no environments"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := dashboard.LoadRecording(strings.NewReader(tc.in)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRecordingWindow(t *testing.T) {
	tests := []struct {
		span, want time.Duration
	}{
		{0, 5 * time.Minute},
		{5 * time.Minute, 5 * time.Minute},
		{11 * time.Minute, 15 * time.Minute},
		{20 * time.Hour, 24 * time.Hour},
		{25*time.Hour + time.Minute, 26 * time.Hour},
	}
	for _, tc := range tests {
		rec := dashboard.Recording{Start: fixedNow(), End: fixedNow().Add(tc.span)}
		if got := rec.Window(); got != tc.want {
			t.Errorf("Window() for %v = %v, want %v", tc.span, got, tc.want)
		}
	}
}
