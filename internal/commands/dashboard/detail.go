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
	writeHeaderLinks(&b, env.Links, w)
	writeBanner(&b, env.Banner, w)
	line("")
	line(statusBar(env, w))
	line(statusLegend(env))
	writeFiring(&b, env, now, w)
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

func writeFiring(b *strings.Builder, env *EnvNode, now time.Time, w int) {
	total := env.FiringCount()
	if total == 0 {
		return
	}
	b.WriteString("\n" + styleBoldRed.Render(fmt.Sprintf("▲ FIRING (%d)", total)) + "\n")
	for _, a := range env.Firing {
		b.WriteString(styleRed.Render(clip(a.Name, w-8)) + " " + styleFaint.Render(Ago(now, a.Since)) + "\n")
		b.WriteString(styleMuted.Render(clip("  "+a.InstanceID, w)) + "\n")
		if a.Message != "" {
			b.WriteString(styleFaint.Render(indent(wrap(a.Message, w-2), "  ")) + "\n")
		}
	}
	b.WriteString(moreAlarms(total, len(env.Firing)))
}

func moreAlarms(total, listed int) string {
	if more := total - listed; more > 0 {
		return styleFaint.Render(fmt.Sprintf("+%d more on the instances' alarm pages", more)) + "\n"
	}
	return ""
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
		badges := instanceBadges(inst, now)
		mark := dot(statusColor(inst.Status))
		if inst.Health != "" {
			mark = dot(healthColor(inst.Health))
		}
		room := w - 2 - lipgloss.Width(badges)
		b.WriteString(mark + " " + clip(name, max(room, 8)) + badges + "\n")
		for _, line := range flow(instanceDetails(inst), w-2) {
			b.WriteString(lipgloss.NewStyle().MaxWidth(w).Render("  "+line) + "\n")
		}
	}
}

func instanceBadges(inst *InstNode, now time.Time) string {
	var tag string
	if inst.Firing > 0 {
		tag += styleRed.Render(fmt.Sprintf(" ▲%d", inst.Firing))
	}
	switch {
	case inst.Deploying():
		since, _ := time.Parse(time.RFC3339, inst.Deploy.CreatedAt)
		tag += styleDeploy.Render(" ⟳ " + strings.ToLower(inst.Deploy.Action) + " " + Ago(now, since))
	case inst.AwaitingApproval():
		tag += styleUpgrade.Render(" ⏸ awaiting approval")
	}
	return tag
}

// flow lays styled parts out in lines no wider than w, joined by dots.
func flow(parts []string, w int) []string {
	var lines []string
	cur := ""
	sep := styleFaint.Render(" · ")
	for _, p := range parts {
		switch {
		case cur == "":
			cur = p
		case lipgloss.Width(cur+sep+p) <= w:
			cur += sep + p
		default:
			lines = append(lines, cur)
			cur = p
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// instanceDetails is what an instance runs and what its widgets say, shown
// under its name.
func instanceDetails(inst *InstNode) []string {
	var parts []string
	if inst.IsApp() {
		parts = append(parts, styleTeal.Render("◆ "+inst.Runtime+" "+shortVersion(inst.RuntimeVersion)))
	}
	if inst.HasUpgrade() {
		parts = append(parts, styleUpgrade.Render("↑"+shortVersion(inst.Upgrade)))
	}
	if inst.Health != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(healthColor(inst.Health)).Render(inst.Health))
	}
	if u := inst.Usage; u != nil {
		parts = append(parts, usageBar(u, 10)+styleMuted.Render(" "+usageText(u)))
	}
	if tr := inst.Traffic; tr != nil {
		text := styleTeal.Render(Rate(tr.Rate) + "/s")
		if tr.Errors > 0 {
			text += styleRed.Render(fmt.Sprintf(" ✕%d in 5m", tr.Errors))
		}
		parts = append(parts, text)
	}
	return parts
}

func usageBar(u *Usage, w int) string {
	filled := min(int(u.Value/u.Max*float64(w)+0.5), w)
	return lipgloss.NewStyle().Foreground(usageColor(u)).Render(strings.Repeat("█", filled)) +
		styleFaint.Render(strings.Repeat("░", w-filled))
}

func usageText(u *Usage) string {
	text := Rate(u.Value) + "/" + Rate(u.Max)
	if u.Label != "" {
		text += " " + u.Label
	}
	return text
}

func writeBanner(b *strings.Builder, banner *Banner, w int) {
	if banner == nil {
		return
	}
	style := lipgloss.NewStyle().Foreground(levelColor(banner.Level))
	b.WriteString("\n" + style.Bold(true).Render("⚑ ") + style.Render(indent(wrap(banner.Message, w-2), "  ")[2:]) + "\n")
}

func writeHeaderLinks(b *strings.Builder, links []HeaderLink, w int) {
	if len(links) == 0 {
		return
	}
	parts := make([]string, 0, len(links))
	for _, l := range links {
		parts = append(parts, hyperlink(l.URL, styleTeal.Render("↗ "+clip(l.Label, w-4))))
	}
	b.WriteString(strings.Join(parts, "\n") + "\n")
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
