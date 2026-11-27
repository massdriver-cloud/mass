package dashboard

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// EnvDetail describes one environment for the side panel, w cells wide.
func EnvDetail(g *Graph, env *EnvNode, blast *Blast, now time.Time, w int) string {
	if env == nil {
		return styleMuted.Render("No environment selected.")
	}
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }

	line(styleTitle.Render(clip(env.Name, w)))
	if env.Restricted {
		line(styleMuted.Render(clip(env.ID, w)))
		line("")
		line(styleMuted.Render(wrap("Instances you can view use resources from this environment, but you cannot view it.", w)))
		return b.String()
	}
	line(styleMuted.Render(clip(env.ProjectName+" · "+env.ID, w)))
	if env.ParentID != "" {
		line(styleFaint.Render(clip("forked from "+env.ParentID, w)))
	}
	if env.Currency != "" {
		line(styleFaint.Render(Money(env.Cost, env.Currency) + " last month"))
	}
	if n := len(env.Deploying()); n > 0 {
		line(styleDeploy.Render(fmt.Sprintf("⟳ %d deploying", n)))
	}
	line("")
	line(statusBar(env, w))
	line(statusLegend(env))
	writeFiring(&b, env.Firing, now, w)
	writeNeighbors(&b, g, "USES RESOURCES FROM", g.Producers(env.ID), func(l Link) string { return l.From }, w)
	writeNeighbors(&b, g, "FEEDS", g.Consumers(env.ID), func(l Link) string { return l.To }, w)
	writeBlast(&b, g, env, blast, w)
	writeInstances(&b, env, now, w)
	return b.String()
}

func statusBar(env *EnvNode, w int) string {
	segs := make([]segment, 0, len(Statuses))
	for _, status := range Statuses {
		segs = append(segs, segment{n: env.StatusCount(status), color: statusColor(status)})
	}
	return bar(segs, w)
}

type segment struct {
	n     int
	color lipgloss.Color
}

// bar draws segments as one w-cell bar, each as wide as its share of the
// total, and never hides a nonzero segment.
func bar(segs []segment, w int) string {
	total := 0
	for _, s := range segs {
		total += s.n
	}
	if total == 0 {
		return styleFaint.Render(strings.Repeat("━", w))
	}
	widths := make([]int, len(segs))
	used, last := 0, 0
	for i, s := range segs {
		if s.n == 0 {
			continue
		}
		widths[i] = max(s.n*w/total, 1)
		used += widths[i]
		last = i
	}
	widths[last] = max(widths[last]+w-used, 1)
	var b strings.Builder
	for i, s := range segs {
		b.WriteString(lipgloss.NewStyle().Foreground(s.color).Render(strings.Repeat("━", widths[i])))
	}
	return b.String()
}

func statusLegend(env *EnvNode) string {
	var parts []string
	for _, status := range Statuses {
		if n := env.StatusCount(status); n > 0 {
			parts = append(parts, dot(statusColor(status))+styleMuted.Render(fmt.Sprintf(" %d %s", n, statusLabel(status))))
		}
	}
	if len(parts) == 0 {
		return styleMuted.Render("No instances")
	}
	return strings.Join(parts, "  ")
}

func writeFiring(b *strings.Builder, firing []FiringAlarm, now time.Time, w int) {
	if len(firing) == 0 {
		return
	}
	b.WriteString("\n" + styleBoldRed.Render(fmt.Sprintf("▲ FIRING (%d)", len(firing))) + "\n")
	for _, a := range firing {
		b.WriteString(styleRed.Render(clip(a.Name, w-8)) + " " + styleFaint.Render(Ago(now, a.Since)) + "\n")
		b.WriteString(styleMuted.Render(clip("  "+a.InstanceID, w)) + "\n")
		if a.Message != "" {
			b.WriteString(styleFaint.Render(indent(wrap(a.Message, w-2), "  ")) + "\n")
		}
	}
}

func writeNeighbors(b *strings.Builder, g *Graph, title string, links []Link, other func(Link) string, w int) {
	if len(links) == 0 {
		return
	}
	b.WriteString("\n" + styleHeading.Render(title) + "\n")
	for _, l := range links {
		b.WriteString(styleTeal.Render("  ⟶ ") + clip(envLabel(g, other(l)), w-10) + styleFaint.Render(fmt.Sprintf(" ×%d", l.Count)) + "\n")
	}
}

func envLabel(g *Graph, id string) string {
	env := g.Env(id)
	if env == nil || env.Restricted || env.ProjectName == "" {
		return id
	}
	return env.ProjectName + " / " + env.Name
}

func writeBlast(b *strings.Builder, g *Graph, env *EnvNode, blast *Blast, w int) {
	if blast == nil {
		return
	}
	envs, insts := blast.Downstream()
	b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(colorOrange).Render("✸ BLAST RADIUS") + "\n")
	if envs == 0 && insts == 0 {
		b.WriteString(styleMuted.Render(wrap("Nothing outside "+env.Name+" uses its resources.", w)) + "\n")
		return
	}
	b.WriteString(styleMuted.Render(fmt.Sprintf("  %d environments · %d instances downstream", envs, insts)) + "\n")
	ids := make([]string, 0, len(blast.Envs))
	for id, hop := range blast.Envs {
		if hop > 0 {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b string) int {
		if blast.Envs[a] != blast.Envs[b] {
			return blast.Envs[a] - blast.Envs[b]
		}
		return strings.Compare(a, b)
	})
	for _, id := range ids {
		hop := blast.Envs[id]
		mark := lipgloss.NewStyle().Foreground(hopColor(hop)).Render(fmt.Sprintf("  %d hop ", hop))
		b.WriteString(mark + clip(envLabel(g, id), w-9) + "\n")
	}
}

func writeInstances(b *strings.Builder, env *EnvNode, now time.Time, w int) {
	if len(env.Instances) == 0 {
		return
	}
	b.WriteString("\n" + styleHeading.Render("INSTANCES") + "\n")
	for _, inst := range env.Instances {
		name := strings.TrimPrefix(inst.ID, env.ID+"-")
		tag := ""
		if inst.IsApp() {
			tag = styleTeal.Render(" ◆ " + inst.Runtime + " " + inst.RuntimeVersion)
		}
		if inst.Firing > 0 {
			tag += styleRed.Render(fmt.Sprintf(" ▲%d", inst.Firing))
		}
		tag += instanceActivity(inst, now)
		room := w - 2 - lipgloss.Width(tag)
		b.WriteString(dot(statusColor(inst.Status)) + " " + clip(name, max(room, 4)) + tag + "\n")
	}
}

func instanceActivity(inst *InstNode, now time.Time) string {
	var tag string
	switch {
	case inst.Deploying():
		since, _ := time.Parse(time.RFC3339, inst.Deploy.CreatedAt)
		tag = styleDeploy.Render(" ⟳ " + strings.ToLower(inst.Deploy.Action) + " " + Ago(now, since))
	case inst.AwaitingApproval():
		tag = styleUpgrade.Render(" ⏸ awaiting approval")
	}
	if inst.HasUpgrade() {
		tag += styleUpgrade.Render(" ↑" + shortVersion(inst.Upgrade))
	}
	return tag
}

// Money formats an amount compactly, in dollars when the currency is USD.
func Money(amount float64, currency string) string {
	var n string
	switch {
	case amount >= 1_000_000:
		n = fmt.Sprintf("%.1fM", amount/1_000_000)
	case amount >= 1000:
		n = fmt.Sprintf("%.1fk", amount/1000)
	default:
		n = fmt.Sprintf("%.0f", amount)
	}
	if currency == "USD" || currency == "" {
		return "$" + n
	}
	return n + " " + currency
}

// Ago says how long before now t was, in the largest whole unit.
func Ago(now, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return runewidth.Truncate(s, w, "…")
}

func wrap(s string, w int) string {
	if w <= 0 {
		return s
	}
	var lines []string
	var cur strings.Builder
	for _, word := range strings.Fields(s) {
		if cur.Len() > 0 && runewidth.StringWidth(cur.String())+1+runewidth.StringWidth(word) > w {
			lines = append(lines, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(word)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return strings.Join(lines, "\n")
}

func indent(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
