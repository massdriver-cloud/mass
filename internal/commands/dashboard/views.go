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
	if m.graph != nil {
		body = overlayFirstLine(body, m.incidentBanner())
	}
	return lipgloss.JoinVertical(lipgloss.Left, fit.Render(m.header()), fit.Render(body), fit.Render(m.ticker()), m.footer())
}

func (m Model) splash(h int) string {
	spin := []string{"◜", "◠", "◝", "◞", "◡", "◟"}[m.frame%6]
	msg := styleTeal.Render(spin) + styleMuted.Render(" charting your constellation…")
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, msg)
}

func (m Model) body(h int) string {
	if m.help && m.tab != TabConstellation {
		return m.helpBox(h, strings.Join(tabHelp(m.tab), "\n"))
	}
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
		Graph: m.graph, Pos: m.animatedPos(), Selected: m.selected, Blast: blast, Theme: m.cfg.Theme,
		ShowApps: m.showApps, GroupByProject: m.grouped, ShowCost: m.showCost, Filter: m.filter,
		Frame: m.frame, Intro: m.intro(), Flashes: m.flashes,
	}
	scene.Target, scene.TargetAge = m.lockOn()
	scene.Now = m.cfg.Now()
	if panelW == 0 && m.help {
		return m.helpBox(h, Legend(m.cfg.Watch, m.snap.AnnotationsErr))
	}
	canvas := overlayFirstLine(scene.Render(canvasW, h), m.bannerStrip(canvasW))
	if panelW == 0 {
		return canvas
	}
	var panel string
	if m.help {
		panel = Legend(m.cfg.Watch, m.snap.AnnotationsErr)
	} else {
		panel = EnvDetail(m.graph, m.graph.Env(m.selected), blast, m.cfg.Now(), panelW)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, canvas, m.panel(panel, panelW, h))
}

// bannerStrip runs the selected environment's banner across the top of the
// map, or else the most urgent banner in view.
func (m Model) bannerStrip(w int) string {
	env := m.graph.Env(m.selected)
	if env == nil || env.Banner == nil {
		env = nil
		rank := map[string]int{"error": 3, "warning": 2}
		best := -1
		for _, e := range m.graph.Envs {
			if e.Banner != nil && rank[e.Banner.Level] > best {
				env, best = e, rank[e.Banner.Level]
			}
		}
	}
	if env == nil {
		return ""
	}
	text := " ⚑ " + envLabel(m.graph, env.ID) + " · " + env.Banner.Message + " "
	return lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(levelColor(env.Banner.Level)).
		MaxWidth(w).Width(w).Render(clip(text, w))
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
	b.WriteString(styleHeading.Render(fmt.Sprintf("FIRING ALARMS IN VIEW (%d)", total)) + "\n\n")
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
	b.WriteString(styleHeading.Render(fmt.Sprintf("UPGRADES AVAILABLE IN VIEW (%d)", len(ups))) + "\n\n")
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

// tabHelp explains the Runtimes, Alarms, and Upgrades tabs for the ? key.
func tabHelp(tab Tab) []string {
	switch tab {
	case TabRuntimes:
		return []string{
			styleHeading.Render("RUNTIMES"), "",
			"Each bar is how the instances in view on a runtime split across its versions. Applications",
			"count across the organization.",
			"",
			note(colorGreen, "━", "on the latest version"),
			note(colorYellow, "━", "a patch behind"),
			note(colorOrange, "━", "a minor behind"),
			note(colorRed, "━", "a major behind"),
			"",
			"p, m, and M show what a patch, minor, or major release would leave behind; 0 goes back.",
		}
	case TabAlarms:
		return []string{
			styleHeading.Render("ALARMS"), "",
			"Each row is an alarm firing now in the environments in view: how long, the alarm, its instance,",
			"and its environment.",
			note(colorOrange, "✸", "how many environments and instances depend on that instance downstream"),
			"",
			"enter shows it on the map with its blast radius; o opens the instance in the web app.",
		}
	case TabUpgrades:
		return []string{
			styleHeading.Render("UPGRADES"), "",
			"Each row is an instance in view whose version constraint allows a newer bundle:",
			"the version it runs → the version its next deploy would use.",
			"",
			"enter shows it on the map; o opens the instance in the web app.",
		}
	case TabConstellation:
	}
	return nil
}

func (m Model) helpBox(h int, content string) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorPanel).Padding(1, 2).
		Width(min(m.width-4, 90)).Render(content)
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, box)
}

func note(color lipgloss.Color, mark, text string) string {
	return lipgloss.NewStyle().Foreground(color).Render(mark) + " " + text
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
			{"c", toggle("cost", m.showCost)}, {"g", toggle("projects", m.grouped)}, {"s", toggle("producers on top", m.stacked)}, {"v", m.cfg.Theme}, {"p", paletteName},
		}...)
	case TabRuntimes:
		keys = append(keys, [2]string{"0/p/m/M", "what if"})
	case TabAlarms, TabUpgrades:
		keys = append(keys, [2]string{"enter", "show on map"})
	}
	if m.cfg.Links != nil {
		keys = append(keys, [2]string{"o", "open"})
	}
	keys = append(keys, [][2]string{{"r", "refresh"}, {"?", "help"}, {"q", "quit"}}...)
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

// Legend explains the marks and motion on the constellation.
func Legend(watch bool, notesErr error) string {
	highlight := lipgloss.NewStyle().Foreground(colorHighlight)
	rows := []string{
		styleHeading.Render("WHAT MOVES"),
		note(colorTeal, "•→", "Sparks: resources flowing to their users"),
		note(colorGreen, "◯", "Ring: the environment's instances by status"),
		note(colorRed, "◉", "Red pulse: an alarm is firing"),
		note(colorBlue, "◜", "Comet around a ring: deploying"),
		highlight.Render("◯") + " Ripple: something just happened",
		highlight.Render("◎") + " Spinning halo: the selected environment",
	}
	if watch {
		rows = append(rows, styleBoldRed.Render("▲")+" Watching: new alarms take the screen")
	}
	rows = append(rows,
		"",
		styleHeading.Render("LINES"),
		styleMuted.Render("━━━ ")+"Uses resources from",
		styleTeal.Render("─ ─ ")+"Forked from",
		styleTeal.Render("1.2k/s")+styleRed.Render(" ✕7")+" Traffic and errors from md/v1/traffic",
		"",
		styleHeading.Render("COLORS"),
		dot(colorGreen)+" Provisioned  "+dot(colorRed)+" Failed",
		dot(colorBlue)+" Not deployed  "+dot(colorGray)+" Decommissioned",
		dot(colorRed)+dot(colorOrange)+dot(colorYellow)+" Blast radius, source to downstream",
		"",
		styleHeading.Render("BADGES"),
		styleBoldRed.Render("▲N")+" alarms firing  "+styleUpgrade.Render("↑N")+" upgrades",
		lipgloss.NewStyle().Foreground(colorOrange).Render("⚑")+" banner  "+styleTeal.Render("◆")+" application  "+styleFaint.Render("◌")+" no access",
		styleNote.Render("✎")+" note",
		"",
		styleHeading.Render("KEYS"),
		keyHint("v", "theme")+"  "+keyHint("p", "palette")+"  "+keyHint("/", "find")+"  "+keyHint("t", "replay")+"  "+keyHint("b", "blast radius"),
		keyHint("a", "apps")+"  "+keyHint("c", "cost")+"  "+keyHint("g", "projects")+"  "+keyHint("s", "producers on top"),
	)
	if notesErr != nil {
		why := "Notes and pinned widgets could not be read: " + clean(notesErr.Error())
		if strings.Contains(notesErr.Error(), "service account") {
			why = "Notes and pinned widgets need a personal access token. This profile signs in as a service account."
		}
		rows = append(rows, "", styleMuted.Render(why))
	}
	return strings.Join(rows, "\n")
}

func keyHint(key, what string) string {
	return styleKey.Render(key) + " " + what
}
