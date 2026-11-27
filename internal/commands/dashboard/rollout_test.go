package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		version string
		bump    dashboard.Bump
		want    string
	}{
		{"1.2.3", dashboard.BumpNone, "1.2.3"},
		{"1.2.3", dashboard.BumpPatch, "1.2.4"},
		{"1.2.3", dashboard.BumpMinor, "1.3.0"},
		{"1.2.3", dashboard.BumpMajor, "2.0.0"},
		{"0.0.0", dashboard.BumpPatch, "0.0.1"},
		{"7.4.33-dev.20261126T223238Z", dashboard.BumpMinor, "7.5.0"},
		{"1.0.0+build", dashboard.BumpPatch, "1.0.1"},
	}
	for _, tc := range tests {
		if got := dashboard.BumpVersion(tc.version, tc.bump); got != tc.want {
			t.Errorf("BumpVersion(%s, %d) = %s, want %s", tc.version, tc.bump, got, tc.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"0.9.0", "1.0.0", -1},
		{"1.10.0", "1.9.0", 1},
		{"1.0.0-dev.1", "1.0.0", -1},
	}
	for _, tc := range tests {
		if got := dashboard.CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestBuildRollout(t *testing.T) {
	snap := fixtureSnapshot(t)
	g := dashboard.BuildGraph(snap)
	rt := snap.Runtimes[0]
	tests := []struct {
		name         string
		bump         dashboard.Bump
		wantLatest   string
		wantOnLatest int
	}{
		{name: "as it is", bump: dashboard.BumpNone, wantLatest: "1.0.0", wantOnLatest: 1},
		{name: "after a patch", bump: dashboard.BumpPatch, wantLatest: "1.0.1", wantOnLatest: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := dashboard.BuildRollout(g, snap, rt, tc.bump)
			if r.Latest != tc.wantLatest || r.Current != "1.0.0" {
				t.Errorf("latest = %s, current = %s", r.Latest, r.Current)
			}
			if got := r.OnLatest(); got != tc.wantOnLatest {
				t.Errorf("on latest = %d, want %d", got, tc.wantOnLatest)
			}
			if len(r.Instances) != 2 || r.Instances[0].ID != "billing-dev-app" {
				t.Errorf("instances = %+v, want the oldest runtime first", r.Instances)
			}
			if len(r.Apps) != 2 {
				t.Errorf("apps = %+v, want the two built on ecs", r.Apps)
			}
			if len(r.Versions) != 2 || r.Versions[0].Version != "1.0.0" {
				t.Errorf("versions = %+v, want newest first", r.Versions)
			}
		})
	}
}

func TestRolloutView(t *testing.T) {
	snap := fixtureSnapshot(t)
	g := dashboard.BuildGraph(snap)
	tests := []struct {
		bump dashboard.Bump
		want []string
	}{
		{dashboard.BumpNone, []string{"ecs latest 1.0.0", "1 of 2 instances run the latest version (1.0.0)", "1.0.0 · 1 (50%)", "Latest", "Behind", "care-app v0.1.0", "Care / Development"}},
		{dashboard.BumpMajor, []string{"Publishing 2.0.0 leaves 2 of 2 instances behind", "M Major 2.0.0"}},
	}
	for _, tc := range tests {
		r := dashboard.BuildRollout(g, snap, snap.Runtimes[0], tc.bump)
		out := ansi.Strip(dashboard.RolloutView(g, r, tc.bump, 120))
		flat := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.want {
			if !strings.Contains(flat, want) {
				t.Errorf("bump %d view is missing %q:\n%s", tc.bump, want, out)
			}
		}
	}
}
