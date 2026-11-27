package dashboard

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/mod/semver"
)

// Bump is a hypothetical next runtime release.
type Bump int

const (
	// BumpNone shows the runtime as it is.
	BumpNone Bump = iota
	// BumpPatch shows what a patch release would leave behind.
	BumpPatch
	// BumpMinor shows what a minor release would leave behind.
	BumpMinor
	// BumpMajor shows what a major release would leave behind.
	BumpMajor
)

// Rollout is how far a runtime's applications and instances are from its
// newest version.
type Rollout struct {
	Runtime   string
	Current   string
	Latest    string
	Instances []RolloutInstance
	Apps      []RolloutApp
	Versions  []VersionShare
}

// RolloutInstance is an instance whose latest provision ran an application on
// the runtime.
type RolloutInstance struct {
	ID             string
	EnvID          string
	App            string
	AppVersion     string
	RuntimeVersion string
}

// RolloutApp is an application built on the runtime.
type RolloutApp struct {
	ID             string
	Version        string
	RuntimeVersion string
	Instances      int
}

// VersionShare is how many instances run one runtime version.
type VersionShare struct {
	Version string
	Count   int
}

// OnLatest counts the instances on the newest version.
func (r Rollout) OnLatest() int {
	n := 0
	for _, inst := range r.Instances {
		if inst.RuntimeVersion == r.Latest {
			n++
		}
	}
	return n
}

// BuildRollout gathers the runtime's instances and applications. With a bump
// it measures them against the version that release would publish.
func BuildRollout(g *Graph, snap *Snapshot, rt Runtime, bump Bump) Rollout {
	r := Rollout{Runtime: rt.ID, Current: rt.Latest.Version, Latest: BumpVersion(rt.Latest.Version, bump)}
	counts := map[string]int{}
	appInstances := map[string]int{}
	for _, env := range g.Envs {
		for _, inst := range env.Instances {
			if inst.Runtime != rt.ID {
				continue
			}
			r.Instances = append(r.Instances, RolloutInstance{
				ID: inst.ID, EnvID: env.ID, App: inst.Bundle, AppVersion: inst.Version, RuntimeVersion: inst.RuntimeVersion,
			})
			counts[inst.RuntimeVersion]++
			appInstances[inst.Bundle]++
		}
	}
	for _, app := range snap.Applications {
		if app.Latest.Runtime == nil || app.Latest.Runtime.Name != rt.ID {
			continue
		}
		r.Apps = append(r.Apps, RolloutApp{
			ID: app.ID, Version: app.Latest.Version, RuntimeVersion: app.Latest.Runtime.Version, Instances: appInstances[app.ID],
		})
	}
	for v, n := range counts {
		r.Versions = append(r.Versions, VersionShare{Version: v, Count: n})
	}
	slices.SortFunc(r.Versions, func(a, b VersionShare) int { return CompareVersions(b.Version, a.Version) })
	slices.SortFunc(r.Instances, func(a, b RolloutInstance) int {
		return cmp.Or(CompareVersions(a.RuntimeVersion, b.RuntimeVersion), cmp.Compare(a.ID, b.ID))
	})
	return r
}

// CompareVersions orders two semantic versions, oldest first.
func CompareVersions(a, b string) int {
	return semver.Compare("v"+a, "v"+b)
}

// BumpVersion is the version a release of the given kind would publish after v.
func BumpVersion(v string, bump Bump) string {
	if bump == BumpNone {
		return v
	}
	core := strings.SplitN(strings.SplitN(v, "-", 2)[0], "+", 2)[0]
	parts := strings.Split(core, ".")
	nums := [3]int{}
	for i := 0; i < 3 && i < len(parts); i++ {
		nums[i], _ = strconv.Atoi(parts[i])
	}
	switch bump {
	case BumpPatch:
		nums[2]++
	case BumpMinor:
		nums[1], nums[2] = nums[1]+1, 0
	case BumpMajor:
		nums[0], nums[1], nums[2] = nums[0]+1, 0, 0
	case BumpNone:
	}
	return fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2])
}

// versionColor shades a version by how far it trails the newest: green on
// the newest, then amber, then red.
func versionColor(v, latest string) lipgloss.Color {
	switch {
	case v == latest:
		return colorGreen
	case semver.Major("v"+v) != semver.Major("v"+latest):
		return colorRed
	case semver.MajorMinor("v"+v) != semver.MajorMinor("v"+latest):
		return colorOrange
	default:
		return colorYellow
	}
}

// RolloutView renders the runtime panel w cells wide.
func RolloutView(g *Graph, r Rollout, bump Bump, w int) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	line(styleTitle.Render(r.Runtime) + styleMuted.Render("  latest "+shortVersion(r.Current)))
	line(styleMuted.Render("Which runtime versions your applications are built on and your instances run."))
	line("")
	line(bumpPicker(r, bump))
	line("")
	total := len(r.Instances)
	headline := fmt.Sprintf("%d of %d instances run the latest version (%s)", r.OnLatest(), total, shortVersion(r.Latest))
	if bump != BumpNone {
		headline = fmt.Sprintf("Publishing %s leaves %d of %d instances behind", r.Latest, total-r.OnLatest(), total)
	}
	line(styleTitle.Render(headline))
	segs := make([]segment, 0, len(r.Versions))
	legend := make([]string, 0, len(r.Versions))
	for _, v := range r.Versions {
		color := versionColor(v.Version, r.Latest)
		segs = append(segs, segment{n: v.Count, color: color})
		legend = append(legend, dot(color)+styleMuted.Render(fmt.Sprintf(" %s · %d (%d%%)", shortVersion(v.Version), v.Count, v.Count*100/max(total, 1))))
	}
	line(bar(segs, w))
	line(strings.Join(legend, "   "))
	line("")
	line(styleHeading.Render("INSTANCES"))
	if len(r.Instances) == 0 {
		line(styleMuted.Render("No instance you can view runs an application on this runtime."))
	}
	for _, inst := range r.Instances {
		line(rolloutInstanceRow(g, inst, r.Latest, w))
	}
	line("")
	line(styleHeading.Render("APPLICATIONS"))
	for _, app := range r.Apps {
		line(rolloutAppRow(app, r.Latest, w))
	}
	return b.String()
}

func bumpPicker(r Rollout, bump Bump) string {
	opts := []struct {
		key  string
		name string
		kind Bump
	}{
		{"0", "Current", BumpNone}, {"p", "Patch", BumpPatch}, {"m", "Minor", BumpMinor}, {"M", "Major", BumpMajor},
	}
	parts := make([]string, 0, len(opts))
	for _, o := range opts {
		label := fmt.Sprintf(" %s %s %s ", o.key, o.name, BumpVersion(r.Current, o.kind))
		if o.kind == BumpNone {
			label = fmt.Sprintf(" %s %s ", o.key, o.name)
		}
		if o.kind == bump {
			parts = append(parts, styleSelected.Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(colorTeal).Render(label))
		}
	}
	return styleMuted.Render("What if you publish? ") + strings.Join(parts, styleFaint.Render("│"))
}

// shortVersion drops the build timestamp from a development version.
func shortVersion(v string) string {
	if i := strings.Index(v, "-dev."); i >= 0 {
		return v[:i+4]
	}
	return v
}

func versionBadge(v, latest string) string {
	if v == latest {
		return styleBadge.Render("Latest")
	}
	return styleBehind.Render("Behind")
}

func rolloutInstanceRow(g *Graph, inst RolloutInstance, latest string, w int) string {
	col := max((w-28)/3, 8)
	row := dot(versionColor(inst.RuntimeVersion, latest)) + " " +
		pad(clip(inst.ID, col), col) + " " +
		styleMuted.Render(pad(clip(envLabel(g, inst.EnvID), col), col)) + " " +
		pad(clip(inst.App+" v"+shortVersion(inst.AppVersion), col), col) + " " +
		"v" + shortVersion(inst.RuntimeVersion) + " " + versionBadge(inst.RuntimeVersion, latest)
	return lipgloss.NewStyle().MaxWidth(w).Render(row)
}

func rolloutAppRow(app RolloutApp, latest string, w int) string {
	col := max((w-24)/2, 10)
	row := "  " + pad(clip(app.ID+" v"+shortVersion(app.Version), col), col) + " " +
		styleMuted.Render("built on ") + "v" + shortVersion(app.RuntimeVersion) + " " + versionBadge(app.RuntimeVersion, latest) +
		styleMuted.Render(fmt.Sprintf("  %d instances", app.Instances))
	return lipgloss.NewStyle().MaxWidth(w).Render(row)
}

func pad(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}
