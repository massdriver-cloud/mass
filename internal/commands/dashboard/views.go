package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View draws the dashboard.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	h := m.bodyHeight()
	var body string
	switch {
	case m.graph == nil && m.err != nil:
		body = styleRed.Render("Could not read the dashboard: " + m.err.Error())
	case m.graph == nil:
		body = m.splash(h)
	default:
		body = m.body(h)
	}
	fit := lipgloss.NewStyle().MaxWidth(m.width)
	body = lipgloss.NewStyle().Width(m.width).Height(h).MaxHeight(h).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, fit.Render(m.header()), fit.Render(body), fit.Render(m.ticker()), m.footer())
}

func (m Model) splash(h int) string {
	spin := []string{"◜", "◠", "◝", "◞", "◡", "◟"}[m.frame%6]
	msg := styleTeal.Render(spin) + styleMuted.Render(" charting your constellation…")
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, msg)
}

func (m Model) body(h int) string {
	switch m.tab {
	case TabRuntimes:
		return m.runtimesBody(h)
	case TabAlarms:
		return m.alarmsBody(h)
	case TabUpgrades:
		return m.upgradesBody(h)
	case TabConstellation:
	}
	return m.constellationBody(h)
}

func (m Model) constellationBody(h int) string {
	panelW := m.panelWidth()
	canvasW, _ := m.canvasSize()
	blast := m.blast()
	scene := Scene{
		Graph: m.graph, Pos: m.pos, Selected: m.selected, Blast: blast,
		ShowApps: m.showApps, GroupByProject: m.grouped, ShowCost: m.showCost, Filter: m.filter,
		Frame: m.frame, Intro: m.intro(), Flashes: m.flashes,
	}
	canvas := scene.Render(canvasW, h)
	if panelW == 0 {
		return canvas
	}
	var panel string
	if m.help {
		panel = Legend()
	} else {
		panel = EnvDetail(m.graph, m.graph.Env(m.selected), blast, m.cfg.Now(), panelW)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, canvas, m.panel(panel, panelW, h))
}

func (m Model) panel(content string, w, h int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorPanel).
		PaddingLeft(1).PaddingRight(1).
		Width(w + 2).Height(h).MaxHeight(h).
		Render(content)
}

func (m Model) runtimesBody(h int) string {
	if len(m.snap.Runtimes) == 0 {
		return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center,
			styleMuted.Render("No runtimes yet. A bundle becomes a runtime when its massdriver.yaml provides `runtime`."))
	}
	listW := 30
	var list strings.Builder
	list.WriteString(styleHeading.Render("RUNTIMES") + "\n\n")
	for i, rt := range m.snap.Runtimes {
		r := BuildRollout(m.graph, m.snap, rt, BumpNone)
		label := fmt.Sprintf(" %-16s %d/%d ", clip(rt.ID, 16), r.OnLatest(), len(r.Instances))
		if i == m.runtimeIdx {
			list.WriteString(styleSelected.Render(pad(label, listW-2)) + "\n")
		} else {
			list.WriteString(pad(label, listW-2) + "\n")
		}
		list.WriteString(" " + rolloutMiniBar(r, listW-4) + "\n\n")
	}
	rt := m.snap.Runtimes[m.runtimeIdx]
	detailW := m.width - listW - 4
	detail := RolloutView(m.graph, BuildRollout(m.graph, m.snap, rt, m.bump), m.bump, detailW)
	left := lipgloss.NewStyle().Width(listW).Height(h).Render(list.String())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, m.panel(detail, detailW, h))
}

func rolloutMiniBar(r Rollout, w int) string {
	segs := make([]segment, 0, len(r.Versions))
	for _, v := range r.Versions {
		segs = append(segs, segment{n: v.Count, color: versionColor(v.Version, r.Latest)})
	}
	return bar(segs, w)
}

func (m Model) alarmsBody(h int) string {
	total := m.graph.FiringCount()
	if total == 0 {
		calm := styleTeal.Render("✓") + styleMuted.Render(" No alarm is firing. All quiet in the constellation.")
		return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, calm)
	}
	var b strings.Builder
	b.WriteString(styleHeading.Render(fmt.Sprintf("FIRING ALARMS (%d)", total)) + "\n\n")
	for i, a := range m.graph.Firing {
		envs, insts := m.graph.BlastRadius(a.InstanceID).Downstream()
		row := fmt.Sprintf("▲ %-5s %-30s %-24s %-28s ✸ %d env · %d inst downstream", Ago(m.cfg.Now(), a.Since),
			clip(a.Name, 30), clip(a.InstanceID, 24), clip(envLabel(m.graph, a.EnvID), 28), envs, insts)
		if i == m.alarmIdx {
			b.WriteString(styleSelected.Render(pad(row, m.width-2)) + "\n")
			b.WriteString(styleMuted.Render(indent(wrap(a.InstanceID+": "+a.Message, m.width-6), "    ")) + "\n\n")
			continue
		}
		b.WriteString(styleRed.Render(row) + "\n")
	}
	b.WriteString("\n" + moreAlarms(total, len(m.graph.Firing)))
	return b.String()
}

func (m Model) upgradesBody(h int) string {
	ups := m.upgrades()
	if len(ups) == 0 {
		calm := styleTeal.Render("✓") + styleMuted.Render(" Every instance runs the newest version its constraint allows.")
		return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, calm)
	}
	var b strings.Builder
	b.WriteString(styleHeading.Render(fmt.Sprintf("UPGRADES AVAILABLE (%d)", len(ups))) + "\n\n")
	for i, inst := range ups {
		from := shortVersion(inst.Deployed)
		if from == "" {
			from = "never deployed"
		}
		row := fmt.Sprintf("↑ %-30s %-28s %-24s %s → %s", clip(inst.ID, 30), clip(envLabel(m.graph, inst.EnvID), 28),
			clip(inst.Bundle, 24), from, shortVersion(inst.Upgrade))
		if i == m.upgradeIdx {
			b.WriteString(styleSelected.Render(pad(row, m.width-2)) + "\n")
			continue
		}
		b.WriteString(styleUpgrade.Render("↑") + row[len("↑"):] + "\n")
	}
	return b.String()
}

func (m Model) header() string {
	title := styleTeal.Bold(true).Render("✦ mass") + styleMuted.Render(" · ") + styleTitle.Render(m.cfg.Org)
	tabs := make([]string, 0, len(tabNames))
	for i, name := range tabNames {
		label := fmt.Sprintf(" %d %s ", i+1, name)
		if m.width < 120 && Tab(i) != m.tab {
			label = fmt.Sprintf(" %d ", i+1)
		}
		if Tab(i) == m.tab {
			tabs = append(tabs, styleSelected.Render(label))
		} else {
			tabs = append(tabs, styleMuted.Render(label))
		}
	}
	left := title + "   " + strings.Join(tabs, " ")
	stats := m.stats()
	for len(stats) > 1 && lipgloss.Width(left)+lipgloss.Width(strings.Join(stats, styleFaint.Render(" · "))) >= m.width {
		stats = stats[1:]
	}
	right := strings.Join(stats, styleFaint.Render(" · "))
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right + "\n"
}

// stats are the header's figures, least important first.
func (m Model) stats() []string {
	if m.graph == nil {
		return nil
	}
	envs, deploying, upgrades := 0, 0, 0
	cost, currency := 0.0, ""
	for _, e := range m.graph.Envs {
		if !e.Restricted {
			envs++
		}
		deploying += len(e.Deploying())
		upgrades += e.Upgrades()
		cost += e.Cost
		if e.Currency != "" {
			currency = e.Currency
		}
	}
	parts := []string{styleMuted.Render(fmt.Sprintf("%d environments · %d projects", envs, len(m.graph.Projects)))}
	if currency != "" {
		parts = append(parts, styleMuted.Render(Money(cost, currency)+"/mo"))
	}
	if upgrades > 0 {
		parts = append(parts, styleUpgrade.Render(fmt.Sprintf("↑ %d", upgrades)))
	}
	if deploying > 0 {
		parts = append(parts, styleDeploy.Render(fmt.Sprintf("⟳ %d deploying", deploying)))
	}
	refresh := styleFaint.Render("⟳ " + Ago(m.cfg.Now(), m.fetched))
	if m.loading {
		refresh = styleTeal.Render("⟳ …")
	}
	return append(parts, refresh, m.alarmStat())
}

func (m Model) alarmStat() string {
	alarming := map[string]bool{}
	for _, a := range m.graph.Firing {
		alarming[a.InstanceID] = true
	}
	if len(alarming) > 0 {
		return styleBoldRed.Render(fmt.Sprintf("%d alarming", len(alarming)))
	}
	return styleTeal.Render("all clear")
}

func (m Model) footer() string {
	if m.searching {
		return styleKey.Render("/") + " " + m.filter + styleTeal.Render("█") + styleFaint.Render("  enter select · esc clear")
	}
	if m.toast != "" && m.frame-m.toastAt < toastFrames {
		return lipgloss.NewStyle().MaxWidth(m.width).Render(styleTeal.Render(m.toast))
	}
	keys := [][2]string{{"←↑↓→", "move"}, {"1-4", "tabs"}}
	switch {
	case m.replay.picking:
		return lipgloss.NewStyle().MaxWidth(m.width).Render(styleKey.Render("←→") + styleMuted.Render(" how far back") +
			styleFaint.Render("  ·  ") + styleKey.Render("enter") + styleMuted.Render(" play") +
			styleFaint.Render("  ·  ") + styleKey.Render("esc") + styleMuted.Render(" cancel"))
	case m.replay.on:
		keys = append(keys, [][2]string{{"t", "stop replay"}, {"space", "pause"}, {"[ ]", "speed"}}...)
	}
	switch m.tab {
	case TabConstellation:
		if !m.replay.on {
			keys = append(keys, [2]string{"t", "replay"})
		}
		keys = append(keys, [][2]string{
			{"/", "find"}, {"b", toggle("blast", m.blastOn)}, {"a", toggle("apps", m.showApps)},
			{"c", toggle("cost", m.showCost)}, {"g", toggle("projects", m.grouped)}, {"s", toggle("producers on top", m.stacked)},
		}...)
	case TabRuntimes:
		keys = append(keys, [2]string{"0/p/m/M", "what if"})
	case TabAlarms, TabUpgrades:
		keys = append(keys, [2]string{"enter", "show on map"})
	}
	if m.cfg.Links != nil {
		keys = append(keys, [2]string{"o", "open"})
	}
	keys = append(keys, [][2]string{{"r", "refresh"}, {"?", "legend"}, {"q", "quit"}}...)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, styleKey.Render(k[0])+" "+styleMuted.Render(k[1]))
	}
	line := strings.Join(parts, styleFaint.Render("  ·  "))
	if m.err != nil && m.graph != nil {
		line = styleRed.Render("refresh failed: "+clip(m.err.Error(), 60)) + "  " + line
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}

func toggle(name string, on bool) string {
	if on {
		return name + " ●"
	}
	return name + " ○"
}

// Legend explains the marks on the constellation.
func Legend() string {
	rows := []string{
		styleHeading.Render("LEGEND"),
		"",
		styleMuted.Render("━━━ ") + "Uses resources from",
		styleTeal.Render("─ ─ ") + "Forked from",
		styleTeal.Render(" •→ ") + "Resources flowing to a consumer",
		"",
		dot(colorGreen) + " Provisioned",
		dot(colorRed) + " Failed",
		dot(colorBlue) + " Not deployed",
		dot(colorGray) + " Decommissioned",
		styleTeal.Render("◆") + " Application",
		"",
		styleBoldRed.Render("▲") + " Alarm firing",
		dot(colorRed) + dot(colorOrange) + dot(colorYellow) + " Blast radius, source to downstream",
		styleDeploy.Render("⟳") + " Deploying",
		styleUpgrade.Render("↑") + " Upgrade available",
		lipgloss.NewStyle().Foreground(colorHighlight).Render("◯") + " Something just happened",
		styleFaint.Render("◌") + " No access",
	}
	return strings.Join(rows, "\n")
}
