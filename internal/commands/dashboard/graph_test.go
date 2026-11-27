package dashboard_test

import (
	"maps"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestBuildGraph(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))

	if got := len(g.Envs); got != 6 {
		t.Errorf("envs = %d, want 5 visible and 1 restricted", got)
	}
	if env := g.Env("secret-prod"); env == nil || !env.Restricted {
		t.Errorf("secret-prod = %+v, want a restricted placeholder", env)
	}
	if got, want := g.Projects, []string{"billing", "care", "platform", "sandbox"}; !equal(got, want) {
		t.Errorf("projects = %v, want %v", got, want)
	}
	if len(g.Firing) != 1 || g.Firing[0].InstanceID != "platform-dev-db" || g.Firing[0].Since.IsZero() {
		t.Errorf("firing = %+v", g.Firing)
	}

	links := map[string]dashboard.LinkKind{}
	for _, l := range g.EnvLinks {
		links[l.From+">"+l.To] = l.Kind
	}
	want := map[string]dashboard.LinkKind{
		"platform-dev>care-dev":         dashboard.LinkDependency,
		"secret-prod>care-dev":          dashboard.LinkDependency,
		"care-dev>billing-dev":          dashboard.LinkDependency,
		"sandbox-template>sandbox-demo": dashboard.LinkFork,
	}
	if !maps.Equal(links, want) {
		t.Errorf("env links = %v, want %v", links, want)
	}

	billing := g.Instance("billing-dev-app")
	if billing.Runtime != "ecs" || billing.RuntimeVersion != "0.9.0" || billing.Version != "1.9.0" || !billing.IsApp() {
		t.Errorf("billing app = %+v, want the bundle from its latest provision", billing)
	}
	if g.Instance("platform-dev-db").Firing != 1 {
		t.Error("platform-dev-db should count its firing alarm")
	}
}

func TestHealth(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	tests := map[string]string{
		"platform-dev":     dashboard.StatusFailed,
		"care-dev":         dashboard.StatusProvisioned,
		"billing-dev":      dashboard.StatusFailed,
		"sandbox-template": dashboard.StatusInitialized,
	}
	for id, want := range tests {
		if got := g.Env(id).Health(); got != want {
			t.Errorf("%s health = %s, want %s", id, got, want)
		}
	}
}

func TestNeighbors(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	if got := g.Producers("care-dev"); len(got) != 2 {
		t.Errorf("care-dev producers = %v, want 2", got)
	}
	if got := g.Consumers("care-dev"); len(got) != 1 || got[0].To != "billing-dev" {
		t.Errorf("care-dev consumers = %v", got)
	}
}

func TestBlastRadius(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	tests := []struct {
		name      string
		sources   []string
		wantEnvs  map[string]int
		wantInsts int
	}{
		{
			name:      "from a firing instance",
			sources:   []string{"platform-dev-db"},
			wantEnvs:  map[string]int{"platform-dev": 0, "care-dev": 1, "billing-dev": 2},
			wantInsts: 2,
		},
		{
			name:      "from a whole environment",
			sources:   []string{"platform-dev"},
			wantEnvs:  map[string]int{"platform-dev": 0, "care-dev": 1, "billing-dev": 2},
			wantInsts: 2,
		},
		{
			name:      "from a leaf",
			sources:   []string{"billing-dev"},
			wantEnvs:  map[string]int{"billing-dev": 0},
			wantInsts: 0,
		},
		{
			name:     "from nothing known",
			sources:  []string{"nope"},
			wantEnvs: map[string]int{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blast := g.BlastRadius(tc.sources...)
			if !maps.Equal(blast.Envs, tc.wantEnvs) {
				t.Errorf("envs = %v, want %v", blast.Envs, tc.wantEnvs)
			}
			if _, insts := blast.Downstream(); insts != tc.wantInsts {
				t.Errorf("downstream instances = %d, want %d", insts, tc.wantInsts)
			}
		})
	}
	var none *dashboard.Blast
	if _, ok := none.Reaches("care-dev"); ok {
		t.Error("a nil blast reaches nothing")
	}
}

func TestLayers(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	want := map[string]int{
		"platform-dev": 0, "secret-prod": 0, "care-dev": 1, "billing-dev": 2, "sandbox-template": 0, "sandbox-demo": 0,
	}
	if got := g.Layers(); !maps.Equal(got, want) {
		t.Errorf("layers = %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestInstanceActivity(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	tests := []struct {
		id                              string
		deploying, awaiting, hasUpgrade bool
	}{
		{id: "billing-dev-app", deploying: true},
		{id: "care-dev-queue", awaiting: true},
		{id: "care-dev-app", hasUpgrade: true},
		{id: "platform-dev-db"},
	}
	for _, tc := range tests {
		inst := g.Instance(tc.id)
		if inst.Deploying() != tc.deploying || inst.AwaitingApproval() != tc.awaiting || inst.HasUpgrade() != tc.hasUpgrade {
			t.Errorf("%s: deploying %v, awaiting %v, upgrade %v; want %v, %v, %v", tc.id,
				inst.Deploying(), inst.AwaitingApproval(), inst.HasUpgrade(), tc.deploying, tc.awaiting, tc.hasUpgrade)
		}
	}
	if got := len(g.Env("billing-dev").Deploying()); got != 1 {
		t.Errorf("billing-dev deploying = %d, want 1", got)
	}
	if got := g.Env("care-dev").Upgrades(); got != 1 {
		t.Errorf("care-dev upgrades = %d, want 1; a never-deployed instance has nothing to upgrade", got)
	}
	if env := g.Env("platform-dev"); env.Cost != 1250 || env.Currency != "USD" {
		t.Errorf("platform-dev cost = %v %s", env.Cost, env.Currency)
	}
}
