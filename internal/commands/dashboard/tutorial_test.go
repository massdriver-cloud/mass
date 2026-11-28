package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func tutorial(t *testing.T) dashboard.Model {
	t.Helper()
	api, cfg := dashboard.Tutorial(dashboard.Config{Now: fixedNow, EventPoll: time.Millisecond})
	return loaded(t, api, cfg, 200, 50)
}

func TestTutorialSteps(t *testing.T) {
	m := tutorial(t)
	tests := []struct {
		title    string
		selected string
		want     []string
	}{
		{"The constellation", "ecomm-prod", []string{"Each ring is an environment", "--depth"}},
		{"Rings", "data-prod", []string{"all four", "provisioned", "failed", "not deployed", "decommissioned"}},
		{"Instances", "ecomm-prod", []string{"larger dots", "md/v1/health", "healthy", "degraded", "down"}},
		{"Alarms", "ecomm-prod", []string{"red core", "▲1", "Alarms tab (3)"}},
		{"Deploying", "ecomm-staging", []string{"comet", "f follows its logs", "⟳ provision"}},
		{"Selection", "network-prod", []string{"spinning halo", "⏸ awaiting approval"}},
		{"Sparks", "data-prod", []string{"md/v1/traffic", "3.4k/s 48ms ✕14"}},
		{"Forks", "ecomm-staging", []string{"dashed line", "forked from ecomm-prod"}},
		{"Blast radius", "ecomm-prod", []string{"alarmed instance", "source", "one hop", "two or more", "BLAST RADIUS"}},
		{"Upgrades", "ecomm-prod", []string{"newer bundle version", "↑1.5.0"}},
		{"Banners", "ecomm-staging", []string{"md/v1/banner", "⚑ Storefront / staging · Load test", "warning"}},
		{"Notes and links", "ecomm-prod", []string{"Only you see your notes", "✎ On call: #storefront-oncall", "↗ Runbook"}},
		{"Out of view", "data-lake", []string{"past the edge", "🔭"}},
		{"No access", "secops-vault", []string{"you cannot view", "no access", "◌"}},
		{"Flashes", "secops-vault", []string{"ripples", "🚀 deploy started", "📌 stash key set", "just now 🚀 Data Platform / ml"}},
		{"Ticker", "secops-vault", []string{"follows the audit log", "LIVE"}},
		{"Side panel", "ecomm-prod", []string{"describes the selected environment", "90 columns"}},
		{"Keys", "ecomm-prod", []string{"1-5 Constellation, Runtimes", "/ find an environment", "w write up an incident", "? every key"}},
	}
	m = press(m, "left")
	for i, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			if i > 0 {
				m = press(m, "right")
			}
			out := view(m)
			counter := fmt.Sprintf("%d/%d", i+1, len(tests))
			for _, want := range append([]string{tc.title, counter}, tc.want...) {
				if !strings.Contains(out, want) {
					t.Errorf("step %s is missing %q:\n%s", counter, want, out)
				}
			}
			if m.Selected() != tc.selected {
				t.Errorf("selected %q, want %q", m.Selected(), tc.selected)
			}
		})
	}
}

func TestTutorialKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		want     string
		quits    bool
		tutorial bool
	}{
		{name: "space and enter go forward", keys: []string{" ", "enter", "right"}, want: "4/18", tutorial: true},
		{name: "left goes back", keys: []string{"right", "right", "left"}, want: "2/18", tutorial: true},
		{name: "left stops at the first", keys: []string{"left", "left"}, want: "1/18", tutorial: true},
		{name: "other keys do nothing", keys: []string{"right", "2", "v", "y"}, want: "2/18", tutorial: true},
		{name: "q quits", keys: []string{"q"}, quits: true, tutorial: true},
		{name: "esc quits", keys: []string{"right", "esc"}, quits: true, tutorial: true},
		{name: "past the last step the map is yours", keys: append(repeat("right", 18), "2"), want: "RUNTIMES"},
	}
	base := tutorial(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, cmd := pressCmd(base, tc.keys...)
			if tc.quits {
				if cmd == nil {
					t.Fatal("no command, want quit")
				}
				if _, ok := cmd().(tea.QuitMsg); !ok {
					t.Error("did not quit")
				}
				return
			}
			out := view(m)
			if !strings.Contains(out, tc.want) {
				t.Errorf("view is missing %q:\n%s", tc.want, out)
			}
			if got := strings.Contains(out, "→ next") || strings.Contains(out, "→ explore"); got != tc.tutorial {
				t.Errorf("tutorial footer shown %v, want %v", got, tc.tutorial)
			}
		})
	}
}

func repeat(key string, n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = key
	}
	return keys
}

func TestTutorialOrganization(t *testing.T) {
	api, cfg := dashboard.Tutorial(dashboard.Config{Now: fixedNow})
	ctx := context.Background()
	snap, err := dashboard.Fetch(ctx, api, dashboard.Where{Focus: cfg.Focus, Depth: cfg.Depth})
	if err != nil {
		t.Fatal(err)
	}
	g := dashboard.BuildGraph(snap)
	g.ApplyStash(dashboard.ReadStash(ctx, api, snap.Environments).Stash, snap.Annotations, fixedNow())

	deploying, upgrades, firing, banners, links := 0, 0, 0, 0, 0
	statuses := map[string]bool{}
	for _, env := range g.Envs {
		deploying += len(env.Deploying())
		upgrades += env.Upgrades()
		firing += env.FiringCount()
		if env.Banner != nil {
			banners++
		}
		for _, inst := range env.Instances {
			statuses[inst.Status] = true
		}
	}
	for _, l := range g.EnvLinks {
		if l.Kind == dashboard.LinkFork || g.Traffic(l) != nil {
			links++
		}
	}
	projects := len(g.Projects)
	if len(g.Envs) < 6 || len(g.Envs) > 9 || projects != 3 {
		t.Errorf("%d environments in %d projects, want 6 to 9 in 3", len(g.Envs), projects)
	}
	for _, status := range dashboard.Statuses {
		if !statuses[status] {
			t.Errorf("no instance is %s", status)
		}
	}
	if deploying != 1 || upgrades != 1 || firing != 1 || banners != 1 || links < 3 {
		t.Errorf("%d deploying, %d upgrades, %d firing, %d banners, %d forks and links with traffic", deploying, upgrades, firing, banners, links)
	}
	if env := g.Env("data-lake"); env == nil || !env.Unloaded {
		t.Errorf("data-lake is %+v, want out of view", env)
	}
	if env := g.Env("secops-vault"); env == nil || !env.Restricted {
		t.Errorf("secops-vault is %+v, want no access", env)
	}
	if snap.AnnotationsErr != nil || len(snap.Annotations) == 0 {
		t.Errorf("annotations %v, %v", snap.Annotations, snap.AnnotationsErr)
	}
}

func TestTutorialWritesNothing(t *testing.T) {
	api, _ := dashboard.Tutorial(dashboard.Config{Now: fixedNow})
	ctx := context.Background()
	for name, err := range map[string]error{
		"approve": api.ApproveDeployment(ctx, "dep-dns"),
		"reject":  api.RejectDeployment(ctx, "dep-dns", "no"),
		"deploy":  api.Deploy(ctx, "ecomm-prod-api"),
	} {
		if !errors.Is(err, dashboard.ErrNoDeployer) {
			t.Errorf("%s: %v, want %v", name, err, dashboard.ErrNoDeployer)
		}
	}
}

func TestTutorialLogs(t *testing.T) {
	api, _ := dashboard.Tutorial(dashboard.Config{Now: fixedNow})
	log, err := api.DeploymentLogs(context.Background(), "dep-staging-api")
	if err != nil || log.Status != "RUNNING" || len(log.Batches) == 0 {
		t.Errorf("logs %+v, %v", log, err)
	}
	if _, err := api.DeploymentLogs(context.Background(), "nope"); err == nil {
		t.Error("logs of an unknown deployment, want an error")
	}
}
