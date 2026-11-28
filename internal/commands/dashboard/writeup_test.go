package dashboard_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
)

type disk struct {
	files map[string]string
	err   error
}

func (d *disk) write(name string, data []byte) error {
	if d.err != nil {
		return d.err
	}
	if d.files == nil {
		d.files = map[string]string{}
	}
	d.files[name] = string(data)
	return nil
}

func (d *disk) names() []string {
	var out []string
	for name := range d.files {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// clocked is a watching config whose clock the test moves.
func clocked(now *time.Time, d *disk) dashboard.Config {
	cfg := config(&opener{})
	cfg.Watch, cfg.Notify, cfg.WriteFile = true, (&notes{}).notify, d.write
	cfg.Now = func() time.Time { return *now }
	return cfg
}

func setAlarm(inst *dashboard.Instance, name, message string) {
	inst.Alarms.Items = nil
	if name != "" {
		inst.Alarms.Items = []dashboard.Alarm{{DisplayName: name, CurrentState: &dashboard.AlarmState{Status: "ALARM", Message: message}}}
	}
}

func careApp(api *fakeAPI) *dashboard.Instance { return &api.envPages[0][1].Instances[0] }

func billingApp(api *fakeAPI) *dashboard.Instance { return &api.envPages[0][2].Instances[0] }

// pollEvents reads the audit log once more, getting log back.
func pollEvents(t *testing.T, api *fakeAPI, m dashboard.Model, log dashboard.Log) dashboard.Model {
	t.Helper()
	api.eventReads = append(make([]dashboard.Log, api.eventCalls), log)
	batch, ok := m.Init()().(tea.BatchMsg)
	if !ok {
		t.Fatal("Init should batch its commands")
	}
	for _, cmd := range batch {
		if msg := cmd(); fmt.Sprintf("%T", msg) == "dashboard.eventsMsg" {
			next, _ := m.Update(msg)
			return next.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
		}
	}
	t.Fatal("Init should read the audit log")
	return m
}

// writeUp presses w and writes what it asks to.
func writeUp(t *testing.T, m dashboard.Model) dashboard.Model {
	t.Helper()
	m, cmd := pressCmd(m, "w")
	if cmd == nil {
		return m
	}
	next, _ := m.Update(cmd())
	return next.(dashboard.Model) //nolint:errcheck,forcetypeassert // Update always returns a Model
}

// incidentScenario fires an alarm on care-dev-app a minute in, with a
// deployment downstream five minutes before it and another while it fires,
// and clears it five minutes later.
func incidentScenario(t *testing.T, d *disk) dashboard.Model {
	t.Helper()
	now := fixedNow()
	api := fixture()
	api.eventReads = []dashboard.Log{{Events: []dashboard.Event{
		{ID: "old", Type: "instance.created", EnvID: "care-dev", At: now.Add(-30 * time.Minute)},
		{ID: "lead", Type: "deployment.completed", EnvID: "billing-dev", At: now.Add(-5 * time.Minute), Detail: "provision", Actor: "ana"},
		{ID: "upstream", Type: "deployment.completed", EnvID: "platform-dev", At: now.Add(-4 * time.Minute)},
	}}}
	release := "v42"
	api.stash = []dashboard.StashEntry{{Scope: dashboard.Scope{Kind: stash.ScopeEnvironment, ID: "care-dev"}, Key: "release", Value: &release}}
	m := loaded(t, api, clocked(&now, d), 200, 48)

	now = now.Add(time.Minute)
	setAlarm(careApp(api), "Latency | p99", "p99 over 2s")
	m = reread(t, m)

	now = now.Add(2 * time.Minute)
	m = pollEvents(t, api, m, dashboard.Log{Events: []dashboard.Event{
		{ID: "elsewhere", Type: "instance.updated", EnvID: "sandbox-demo", At: now},
		{ID: "during", Type: "deployment.failed", EnvID: "billing-dev", At: now, Actor: "bo"},
	}})

	now = now.Add(3 * time.Minute)
	setAlarm(careApp(api), "", "")
	return reread(t, m)
}

func TestWriteup(t *testing.T) {
	d := &disk{}
	m := incidentScenario(t, d)
	m = writeUp(t, m)
	name := "incident-care-dev-app-latency-p99-20261126-0001"
	if got := d.names(); !slices.Equal(got, []string{name + ".json", name + ".md"}) {
		t.Fatalf("wrote %v", got)
	}
	if out := view(m); !strings.Contains(out, "✎ wrote "+name+".md") {
		t.Errorf("toast should name the write-up:\n%s", out)
	}
	md := d.files[name+".md"]
	for _, want := range []string{
		"# Latency \\| p99 on care-dev-app\n",
		"| Environment | Care / Development (`care-dev`) |",
		"| Started | 2026-11-26 00:01:00 UTC |",
		"| Resolved | 2026-11-26 00:06:00 UTC |",
		"| Duration | 5:00 |",
		"**Latency \\| p99**\n\n> p99 over 2s\n",
		"1 environment and 1 instance downstream of `care-dev-app`:\n\n- Billing / Development: `billing-dev-app` (1 hop)\n",
		"| 2026-11-25 23:55:00 | deployment completed | Billing / Development | provision | ana |\n" +
			"| 00:01:00 | alarm fired | Care / Development | Latency \\| p99 |  |\n" +
			"| 00:03:00 | deployment failed | Billing / Development |  | bo |\n" +
			"| 00:06:00 | alarm cleared | Care / Development | Latency \\| p99 |  |\n",
		"## Root cause",
		"## Follow-ups",
		"mass dashboard --playback " + name + ".json",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("write-up is missing %q:\n%s", want, md)
		}
	}
	for _, unwanted := range []string{"instance created", "Platform / Development", "sandbox", "instance updated"} {
		if strings.Contains(md, unwanted) {
			t.Errorf("write-up should hold only the blast radius from the lead-up on, not %q:\n%s", unwanted, md)
		}
	}
}

func TestWriteupPicks(t *testing.T) {
	tests := []struct {
		name  string
		keys  []string
		want  []string
		toast string
	}{
		{
			name: "a firing alarm with no incident yet",
			keys: []string{"3"},
			want: []string{
				"# CPU high on platform-dev-db",
				"| Resolved | still firing at 2026-11-26 00:00:00 UTC |",
				"| Duration | 24:00:00 so far |",
				"2 environments and 2 instances downstream of `platform-dev-db`",
				"- Care / Development: `care-dev-app` (1 hop)\n- Billing / Development: `billing-dev-app` (2 hops)\n",
				"mass dashboard --playback incident-platform-dev-db-cpu-high-20261125-0000.json",
			},
			toast: "✎ wrote incident-platform-dev-db-cpu-high-20261125-0000.md",
		},
		{name: "an environment with nothing firing", toast: "no incident to write up"},
		{name: "another tab", keys: []string{"2"}, toast: "no incident to write up"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &disk{}
			cfg := config(&opener{})
			cfg.WriteFile = d.write
			m := writeUp(t, press(loaded(t, fixture(), cfg, 200, 48), tc.keys...))
			md := ""
			for name, data := range d.files {
				if strings.HasSuffix(name, ".md") {
					md = data
				}
			}
			for _, want := range tc.want {
				if !strings.Contains(md, want) {
					t.Errorf("write-up is missing %q:\n%s", want, md)
				}
			}
			if len(tc.want) == 0 && len(d.files) > 0 {
				t.Errorf("wrote %v", d.names())
			}
			if out := view(m); !strings.Contains(out, tc.toast) {
				t.Errorf("toast should say %q:\n%s", tc.toast, out)
			}
		})
	}
}

func TestWriteupFails(t *testing.T) {
	d := &disk{err: errors.New("disk full")}
	cfg := config(&opener{})
	cfg.WriteFile = d.write
	m := writeUp(t, press(loaded(t, fixture(), cfg, 200, 48), "3"))
	if out := view(m); !strings.Contains(out, "could not write incident-platform-dev-db-cpu-high-20261125-0000.json: disk full") {
		t.Errorf("a failed write should say so:\n%s", out)
	}
}

func TestWriteupHelp(t *testing.T) {
	m := loaded(t, fixture(), config(&opener{}), 200, 48)
	for _, tc := range []struct{ keys []string }{{keys: []string{"?"}}, {keys: []string{"3", "?"}}} {
		if out := view(press(m, tc.keys...)); !strings.Contains(out, "w write") {
			t.Errorf("help after %v should name w:\n%s", tc.keys, out)
		}
	}
}
