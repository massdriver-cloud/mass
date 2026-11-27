package dashboard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestEnvDetail(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		env   string
		blast *dashboard.Blast
		want  []string
	}{
		{
			name:  "firing producer",
			env:   "platform-dev",
			blast: g.BlastRadius("platform-dev-db"),
			want: []string{
				"Development", "Platform · platform-dev", "$1.2k last month", "2 provisioned", "FIRING (1)", "CPU high", "1d", "CPU over 80%",
				"FEEDS", "Care / Development", "BLAST RADIUS", "2 environments · 2 instances", "2 hop Billing / Development",
				"INSTANCES", "network", "db ▲1",
			},
		},
		{
			name: "consumer of a hidden environment",
			env:  "care-dev",
			want: []string{"USES RESOURCES FROM", "secret-prod", "Platform / Development", "app ◆ ecs 1.0.0 ↑0.2.0", "1 not deployed", "queue ⏸ awaiting approval"},
		},
		{
			name:  "leaf",
			env:   "billing-dev",
			blast: g.BlastRadius("billing-dev"),
			want:  []string{"Nothing outside Development uses its resources.", "1 failed", "⟳ 1 deploying", "app ◆ ecs 0.9.0 ⟳ provision 2m"},
		},
		{name: "fork", env: "sandbox-demo", want: []string{"forked from sandbox-template", "No instances"}},
		{name: "restricted", env: "secret-prod", want: []string{"secret-prod", "you cannot view it"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := ansi.Strip(dashboard.EnvDetail(g, g.Env(tc.env), tc.blast, now, 50))
			flat := strings.Join(strings.Fields(out), " ")
			for _, want := range tc.want {
				if !strings.Contains(flat, want) {
					t.Errorf("detail is missing %q:\n%s", want, out)
				}
			}
		})
	}
	if out := ansi.Strip(dashboard.EnvDetail(g, nil, nil, now, 40)); !strings.Contains(out, "No environment selected") {
		t.Errorf("nil env = %q", out)
	}
}

func TestEnvDetailMoreAlarms(t *testing.T) {
	snap := fixtureSnapshot(t)
	snap.Environments[0].Instances[1].FiringAlarmCount = 3
	g := dashboard.BuildGraph(snap)
	out := ansi.Strip(dashboard.EnvDetail(g, g.Env("platform-dev"), nil, time.Now(), 60))
	for _, want := range []string{"FIRING (3)", "CPU high", "+2 more on the instances' alarm pages"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail is missing %q:\n%s", want, out)
		}
	}

	api := fixture()
	api.envPages[0][0].Instances[1].FiringAlarmCount = 3
	tab := view(press(loaded(t, api, config(&opener{}), 200, 48), "3"))
	for _, want := range []string{"FIRING ALARMS (3)", "CPU high", "+2 more on the instances' alarm pages"} {
		if !strings.Contains(tab, want) {
			t.Errorf("alarms tab is missing %q:\n%s", want, tab)
		}
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		since time.Time
		want  string
	}{
		{time.Time{}, ""},
		{now.Add(-10 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-50 * time.Hour), "2d"},
	}
	for _, tc := range tests {
		if got := dashboard.Ago(now, tc.since); got != tc.want {
			t.Errorf("Ago(%v) = %q, want %q", tc.since, got, tc.want)
		}
	}
}

func TestMoney(t *testing.T) {
	tests := []struct {
		amount   float64
		currency string
		want     string
	}{
		{12, "USD", "$12"},
		{1250, "USD", "$1.2k"},
		{3_400_000, "", "$3.4M"},
		{980, "EUR", "980 EUR"},
	}
	for _, tc := range tests {
		if got := dashboard.Money(tc.amount, tc.currency); got != tc.want {
			t.Errorf("Money(%v, %s) = %s, want %s", tc.amount, tc.currency, got, tc.want)
		}
	}
}
