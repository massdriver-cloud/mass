package dashboard_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestTimelineKeepsTheEarliestEvents(t *testing.T) {
	tests := []struct {
		name    string
		events  int
		last    string
		dropped string
		notKept string
	}{
		{name: "under the cap", events: 10, last: "e009"},
		{name: "over the cap", events: 250, last: "e199", dropped: "Not listed: 50 events later in the blast radius.", notKept: "e200"},
		{name: "one over", events: 201, last: "e199", dropped: "Not listed: 1 event later in the blast radius.", notKept: "e200"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now, d := fixedNow(), &disk{}
			api := fixture()
			m := loaded(t, api, clocked(&now, d), 200, 48)
			setAlarm(careApp(api), "Latency", "")
			m = reread(t, m)
			events := make([]dashboard.Event, tc.events)
			for i := range events {
				events[i] = dashboard.Event{ID: strconv.Itoa(i), Type: "deployment.started", EnvID: "billing-dev", At: now.Add(time.Duration(i) * time.Second), Detail: fmt.Sprintf("e%03d", i)}
			}
			m = pollEvents(t, api, m, dashboard.Log{Events: events})
			writeUp(t, m)
			md := d.files["incident-care-dev-app-latency-20261126-0000.md"]
			if !strings.Contains(md, tc.last) || !strings.Contains(md, "e000") {
				t.Errorf("write-up should run from e000 to %s:\n%s", tc.last, md)
			}
			if tc.notKept != "" && strings.Contains(md, tc.notKept) {
				t.Errorf("write-up should stop before %s", tc.notKept)
			}
			if tc.dropped != "" && !strings.Contains(md, tc.dropped) {
				t.Errorf("write-up should say %q:\n%s", tc.dropped, md)
			}
			if tc.dropped == "" && strings.Contains(md, "Not listed") {
				t.Errorf("nothing was left out:\n%s", md)
			}
		})
	}
}

func TestTimelineKeepsRecentIncidents(t *testing.T) {
	tests := []struct {
		name  string
		later int
		file  string
		want  string
	}{
		{name: "the first is still kept", later: 19, file: "incident-care-dev-queue-queue-backed-up-20261126-0000.md", want: "# Queue backed up on care-dev-queue"},
		{name: "the first has been let go", later: 20, file: "incident-billing-dev-app-billing-slow-20261126-0000.md", want: "# Billing slow on billing-dev-app"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now, d := fixedNow(), &disk{}
			api := fixture()
			m := loaded(t, api, clocked(&now, d), 200, 48)
			setFiring(api, true)
			m = reread(t, m)
			setFiring(api, false)
			m = reread(t, m)
			for range tc.later {
				setAlarm(billingApp(api), "Billing slow", "")
				m = reread(t, m)
				setAlarm(billingApp(api), "", "")
				m = reread(t, m)
			}
			m = settle(t, m)
			if m.Selected() != "care-dev" {
				t.Fatalf("selected %s, want the view back on care-dev", m.Selected())
			}
			writeUp(t, m)
			if md := d.files[tc.file]; !strings.Contains(md, tc.want) {
				t.Errorf("wrote %v; want %q", d.names(), tc.want)
			}
		})
	}
}
