package dashboard

import (
	"encoding/xml"
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const svgFonts = "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"

// ExportSVG draws the scene as a w×h-cell map would show it, as an SVG
// picture headed by title.
func ExportSVG(scene Scene, w, h int, title string) string {
	scene.Intro, scene.Flashes, scene.Target = 0, nil, ""
	w, h = max(w, 10), max(h, 3)
	dotW, dotH := float64(w*2), float64(h*4)
	k := math.Min(math.Max(1600/dotW, 4), 10)
	x := &svg{scene: scene, k: k, top: 14 * k, dotW: dotW, dotH: dotH}
	width, height := dotW*k, dotH*k+x.top+10*k
	fmt.Fprintf(&x.b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" font-family="%s">`+"\n",
		width, height, width, height, svgFonts)
	fmt.Fprintf(&x.b, "<title>%s</title>\n", esc(title))
	fmt.Fprintf(&x.b, `<rect width="100%%" height="100%%" fill="%s"/>`+"\n", colorInk)
	x.backdrop()
	x.heading(title)
	fmt.Fprintf(&x.b, `<g transform="translate(0 %.1f)">`+"\n", x.top)
	switch scene.Theme {
	case ThemeMetro:
		x.metro()
	case ThemeRadar:
		x.radar()
	default:
		x.constellation()
	}
	x.b.WriteString("</g>\n")
	x.legend(height - 5*k)
	x.b.WriteString("</svg>\n")
	return x.b.String()
}

type svg struct {
	b          strings.Builder
	scene      Scene
	k, top     float64
	dotW, dotH float64
}

func esc(s string) string {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return ""
	}
	return b.String()
}

func (x *svg) px(p Point) (float64, float64) { return p.X * x.k, p.Y * x.k }

func (x *svg) font(scale float64) float64 { return x.k * scale }

func (x *svg) backdrop() {
	fmt.Fprintf(&x.b, `<g opacity="0.8" transform="translate(0 %.1f)">`+"\n", x.top)
	switch paletteName {
	case "blueprint":
		x.grid()
	case "rain":
		x.rain()
	default:
		x.stars()
	}
	x.b.WriteString("</g>\n")
}

func (x *svg) grid() {
	for gx := 0; gx <= int(x.dotW); gx += 16 {
		x.line(float64(gx), 0, float64(gx), x.dotH, colorStar, gx%64 == 0)
	}
	for gy := 0; gy <= int(x.dotH); gy += 16 {
		x.line(0, float64(gy), x.dotW, float64(gy), colorStar, gy%64 == 0)
	}
}

func (x *svg) rain() {
	for gx := 0; gx < int(x.dotW); gx += 3 {
		if hash01(gx, 0, 11) > 0.2 {
			continue
		}
		head := hash01(gx, 3, 19) * x.dotH
		length := 8 + hash01(gx, 2, 17)*30
		fmt.Fprintf(&x.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f" stroke-linecap="round"/>`+"\n",
			float64(gx)*x.k, (head-length)*x.k, float64(gx)*x.k, head*x.k, colorStar, x.k*0.5)
	}
}

func (x *svg) stars() {
	for gy := range int(x.dotH) {
		for gx := range int(x.dotW) {
			if hash01(gx, gy, 7) > 0.0025 {
				continue
			}
			color, r := colorStar, 0.35
			if hash01(gx, gy, 0) < 0.25 {
				color, r = colorStarLit, 0.5
			}
			fmt.Fprintf(&x.b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", float64(gx)*x.k, float64(gy)*x.k, r*x.k, color)
		}
	}
}

func (x *svg) line(x0, y0, x1, y1 float64, color lipgloss.Color, bold bool) {
	width, dash := x.k*0.15, ""
	if bold {
		width, dash = x.k*0.25, fmt.Sprintf(` stroke-dasharray="%.1f %.1f"`, x.k, x.k)
	}
	fmt.Fprintf(&x.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.2f"%s/>`+"\n",
		x0*x.k, y0*x.k, x1*x.k, y1*x.k, color, width, dash)
}

func (x *svg) heading(title string) {
	fs := x.font(4.2)
	fit := ""
	if room := x.dotW*x.k - 8*x.k; float64(runewidth.StringWidth(title))*fs*0.62 > room {
		fit = fmt.Sprintf(` textLength="%.1f" lengthAdjust="spacingAndGlyphs"`, room)
	}
	fmt.Fprintf(&x.b, `<text x="%.1f" y="%.1f" font-size="%.1f" font-weight="bold" fill="%s"%s>%s</text>`+"\n",
		4*x.k, 9*x.k, fs, colorText, fit, esc(title))
	fmt.Fprintf(&x.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"/>`+"\n",
		4*x.k, 12*x.k, x.dotW*x.k-4*x.k, 12*x.k, colorPanel, x.k*0.2)
}

func (x *svg) legend(y float64) {
	entries := []struct {
		color lipgloss.Color
		label string
	}{
		{colorGreen, "provisioned"}, {colorRed, "failed"}, {colorBlue, "not deployed"}, {colorGray, "decommissioned"},
		{colorOrange, "blast radius"}, {colorYellow, "↑ upgrades"},
	}
	cx, fs := 4*x.k, x.font(2.8)
	fixed, text := 7*x.k*float64(len(entries)), 0.0
	for _, e := range entries {
		text += float64(runewidth.StringWidth(e.label)) * fs * 0.62
	}
	if room := x.dotW*x.k - 8*x.k; fixed+text > room && room > fixed {
		fs *= (room - fixed) / text
	}
	for _, e := range entries {
		fmt.Fprintf(&x.b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`, cx+x.k, y-fs*0.35, x.k*0.9, e.color)
		fmt.Fprintf(&x.b, `<text x="%.1f" y="%.1f" font-size="%.1f" fill="%s">%s</text>`+"\n", cx+3*x.k, y, fs, colorMuted, esc(e.label))
		cx += 3*x.k + float64(runewidth.StringWidth(e.label))*fs*0.62 + 4*x.k
	}
}

func (x *svg) circle(c Point, r float64, color lipgloss.Color, width float64, extra string) {
	cx, cy := x.px(c)
	fmt.Fprintf(&x.b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="%.1f"%s/>`+"\n",
		cx, cy, r*x.k, color, width*x.k, extra)
}

func (x *svg) disc(c Point, r float64, color lipgloss.Color) {
	cx, cy := x.px(c)
	fmt.Fprintf(&x.b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", cx, cy, r*x.k, color)
}

func (x *svg) ring(c Point, r, width float64, colorAt func(float64) lipgloss.Color) {
	const steps = 240
	start, color := 0, colorAt(0)
	for i := 1; i <= steps; i++ {
		next := lipgloss.Color("")
		if i < steps {
			next = colorAt(float64(i) / steps)
		}
		if i < steps && next == color {
			continue
		}
		if color != "" {
			x.arc(c, r, width, float64(start)/steps, float64(i)/steps, color)
		}
		start, color = i, next
	}
}

func (x *svg) arc(c Point, r, width, from, to float64, color lipgloss.Color) {
	if to-from >= 1 {
		x.circle(c, r, color, width, "")
		return
	}
	at := func(f float64) (float64, float64) {
		a := f*2*math.Pi - math.Pi/2
		return (c.X + r*math.Cos(a)) * x.k, (c.Y + r*math.Sin(a)) * x.k
	}
	x0, y0 := at(from)
	x1, y1 := at(to)
	large := 0
	if to-from > 0.5 {
		large = 1
	}
	fmt.Fprintf(&x.b, `<path d="M%.1f %.1f A%.1f %.1f 0 %d 1 %.1f %.1f" fill="none" stroke="%s" stroke-width="%.1f"/>`+"\n",
		x0, y0, r*x.k, r*x.k, large, x1, y1, color, width*x.k)
}

func (x *svg) text(p Point, dx, dy float64, s string, size float64, color lipgloss.Color, anchor string, bold bool) {
	x.textOn(p, dx, dy, s, size, color, anchor, bold, colorInk)
}

// textOn outlines the text in the color behind it, so the links it crosses
// break around it the way they do under text in the terminal.
func (x *svg) textOn(p Point, dx, dy float64, s string, size float64, color lipgloss.Color, anchor string, bold bool, behind lipgloss.Color) {
	px, py := x.px(p)
	attrs := fmt.Sprintf(` stroke="%s" stroke-width="%.1f" stroke-linejoin="round" paint-order="stroke"`, behind, x.k*1.1)
	if bold {
		attrs += ` font-weight="bold"`
	}
	fmt.Fprintf(&x.b, `<text x="%.1f" y="%.1f" font-size="%.1f" fill="%s" text-anchor="%s"%s>%s</text>`+"\n",
		px+dx*x.k, py+dy*x.k, size, color, anchor, attrs, esc(s))
}

func (x *svg) nameColors(env *EnvNode) (lipgloss.Color, lipgloss.Color, bool) {
	s := x.scene
	sub := colorMuted
	if s.GroupByProject {
		sub = projectColor(s.Graph, env.ProjectID)
	}
	switch {
	case env.ID == s.Selected:
		return colorInk, sub, true
	case !s.Matches(env) || env.Restricted || env.Unloaded:
		return colorFaint, colorFaint, false
	case env.FiringCount() > 0:
		return colorRed, sub, false
	}
	return colorText, sub, false
}

func (x *svg) label(env *EnvNode, p Point, dx, dy float64, anchor string, extra string) {
	name, sub, selected := x.nameColors(env)
	fs := x.font(3)
	if selected {
		w := float64(runewidth.StringWidth(env.Name))*fs*0.62 + 2*x.k
		px, py := x.px(p)
		left := px + dx*x.k - x.k
		if anchor == "middle" {
			left = px + dx*x.k - w/2
		}
		fmt.Fprintf(&x.b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s"/>`+"\n",
			left, py+dy*x.k-fs, w, fs*1.35, x.k*0.8, colorTeal)
	}
	behind := colorInk
	if selected {
		behind = colorTeal
	}
	x.textOn(p, dx, dy, env.Name, fs, name, anchor, true, behind)
	x.text(p, dx, dy+3.6, x.scene.subLabel(env)+extra, x.font(2.6), sub, anchor, false)
}

func (x *svg) constellation() {
	s := x.scene
	for _, l := range s.Graph.EnvLinks {
		p0, p1, p2, ok := s.curve(l)
		if !ok {
			continue
		}
		color, layer := s.linkColor(l)
		width, dash := 0.35, ""
		if layer == layerHotLinks {
			width = 0.6
		}
		if l.Kind == LinkFork {
			dash = fmt.Sprintf(` stroke-dasharray="%.1f %.1f"`, x.k*1.5, x.k*1.5)
		}
		fmt.Fprintf(&x.b, `<path d="M%.1f %.1f Q%.1f %.1f %.1f %.1f" fill="none" stroke="%s" stroke-width="%.1f" stroke-linecap="round"%s/>`+"\n",
			p0.X*x.k, p0.Y*x.k, p1.X*x.k, p1.Y*x.k, p2.X*x.k, p2.Y*x.k, color, width*x.k, dash)
	}
	x.sparks()
	for _, env := range s.Graph.Envs {
		x.node(env)
	}
	x.trafficLabels()
}

func (x *svg) sparks() {
	s := x.scene
	for i, l := range s.Graph.EnvLinks {
		p0, p1, p2, ok := s.curve(l)
		if l.Kind != LinkDependency || !ok {
			continue
		}
		color, _ := s.linkColor(l)
		if color == colorLink || color == colorLinkDim {
			color = colorTeal
		}
		n, _, errs := sparkFlow(l, s.Graph.Traffic(l))
		for j := range n {
			t := math.Mod((float64(j)+0.5)/float64(n)+float64(i)*0.137, 1)
			spark := color
			if j < errs {
				spark = errorColor(s.Graph.Traffic(l).Errors)
			}
			x.disc(Bezier(p0, p1, p2, t), 0.7, spark)
		}
	}
}

func (x *svg) node(env *EnvNode) {
	s := x.scene
	c, ok := s.Pos[env.ID]
	if !ok {
		return
	}
	r := s.radius(env)
	x.disc(c, r+0.8, colorInk)
	if s.ShowApps {
		x.satellites(env, c, r)
	}
	x.ring(c, r, 0.9, s.ringColors(env))
	if env.FiringCount() > 0 {
		x.disc(c, 1.5, colorRed)
		x.circle(c, r+2.5, colorRed, 0.4, "")
		x.circle(c, r+5, colorRed, 0.3, ` opacity="0.45"`)
	}
	if len(env.Deploying()) > 0 {
		x.ring(c, r+1.4, 0.5, func(f float64) lipgloss.Color {
			if f < 0.3 {
				return colorBlue
			}
			return ""
		})
	}
	if hop, ok := s.Blast.Reaches(env.ID); ok {
		x.circle(c, r+2.5, hopColor(hop), 0.5, "")
	}
	if env.ID == s.Selected {
		x.circle(c, r+4, colorTeal, 0.45, "")
		x.circle(c, r+4, colorHighlight, 0.45, fmt.Sprintf(` stroke-dasharray="%.1f %.1f"`, x.k*2, x.k*2))
	}
	x.badges(env, c, r)
	x.label(env, c, 0, r+8, "middle", "")
}

func (x *svg) satellites(env *EnvNode, c Point, r float64) {
	s := x.scene
	n := len(env.Instances)
	orbit := r + satelliteRadius - nodeRadius
	for i, inst := range env.Instances {
		angle := 2*math.Pi*float64(i)/float64(n) - math.Pi/2 + 0.3
		p := Point{X: c.X + orbit*1.4*math.Cos(angle), Y: c.Y + orbit*math.Sin(angle)}
		size := 0.6
		if inst.IsApp() {
			size = 1
		}
		x.disc(p, size, s.instanceColor(env, inst))
	}
}

func (x *svg) badges(env *EnvNode, c Point, r float64) {
	fs := x.font(2.8)
	if n := env.FiringCount(); n > 0 {
		x.text(c, r+2, -r, fmt.Sprintf("▲%d", n), fs, colorRed, "start", true)
	}
	if n := env.Upgrades(); n > 0 {
		x.text(c, -r-2, -r, fmt.Sprintf("↑%d", n), fs, colorYellow, "end", true)
	}
	if env.Banner != nil {
		x.text(c, r+2, r+1, "⚑", fs, levelColor(env.Banner.Level), "start", true)
	}
}

func (x *svg) trafficLabels() {
	s := x.scene
	for _, l := range s.Graph.EnvLinks {
		tr := s.Graph.Traffic(l)
		p0, p1, p2, ok := s.curve(l)
		if tr == nil || !ok {
			continue
		}
		label := Rate(tr.Rate) + "/s" + latencyText(tr)
		color := colorTealDim
		if tr.Errors > 0 {
			label += fmt.Sprintf(" ✕%d", tr.Errors)
			color = errorColor(tr.Errors)
		}
		if tr.Stale(s.Now) {
			label += " stale"
			color = colorFaint
		}
		x.text(Bezier(p0, p1, p2, 0.5), 0, -1.5, label, x.font(2.4), color, "middle", false)
	}
}

func cellCenter(cx, cy int) Point { return Point{X: float64(cx*2 + 1), Y: float64(cy*4 + 2)} }

func (x *svg) metro() {
	s := x.scene
	for i, l := range s.Graph.EnvLinks {
		from, ok0 := s.Pos[l.From]
		to, ok1 := s.Pos[l.To]
		if !ok0 || !ok1 {
			continue
		}
		color, _ := s.linkColor(l)
		if color == colorLink || color == colorTealDim {
			color = projectColor(s.Graph, s.Graph.Env(l.From).ProjectID)
		}
		points := []string{fmt.Sprintf("%.1f,%.1f", from.X*x.k, from.Y*x.k)}
		for _, cell := range metroRoute(from, to, i) {
			p := cellCenter(cell[0], cell[1])
			points = append(points, fmt.Sprintf("%.1f,%.1f", p.X*x.k, p.Y*x.k))
		}
		points = append(points, fmt.Sprintf("%.1f,%.1f", to.X*x.k, to.Y*x.k))
		dash := ""
		if l.Kind == LinkFork {
			dash = fmt.Sprintf(` stroke-dasharray="%.1f %.1f"`, x.k*1.2, x.k*1.2)
		}
		fmt.Fprintf(&x.b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-linejoin="round" stroke-linecap="round"%s/>`+"\n",
			strings.Join(points, " "), color, x.k*0.9, dash)
	}
	for _, env := range s.Graph.Envs {
		p, ok := s.Pos[env.ID]
		if !ok {
			continue
		}
		_, color := s.stationGlyph(env)
		x.disc(p, 1.8, colorInk)
		x.circle(p, 1.4, color, 0.8, "")
		if env.ID == s.Selected {
			x.circle(p, 3, colorTeal, 0.4, "")
		}
		x.label(env, p, 5, 1, "start", stationBadges(env))
	}
}

func stationBadges(env *EnvNode) string {
	out := ""
	if n := env.FiringCount(); n > 0 {
		out += fmt.Sprintf(" ▲%d", n)
	}
	if n := env.Upgrades(); n > 0 {
		out += fmt.Sprintf(" ↑%d", n)
	}
	if env.Banner != nil {
		out += " ⚑"
	}
	return out
}

func (x *svg) radar() {
	s := x.scene
	center := Point{X: x.dotW / 2, Y: x.dotH / 2}
	rx, ry := math.Max(x.dotW/2-24, 8), math.Max(x.dotH/2-8, 4)
	cx, cy := x.px(center)
	for _, frac := range []float64{1.0 / 3, 2.0 / 3, 1} {
		fmt.Fprintf(&x.b, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="none" stroke="%s" stroke-width="%.1f"/>`+"\n",
			cx, cy, rx*frac*x.k, ry*frac*x.k, colorStarLit, x.k*0.2)
	}
	x.line(center.X-rx, center.Y, center.X+rx, center.Y, colorStar, true)
	x.line(center.X, center.Y-ry, center.X, center.Y+ry, colorStar, true)
	for _, l := range s.Graph.EnvLinks {
		from, ok0 := s.Pos[l.From]
		to, ok1 := s.Pos[l.To]
		if !ok0 || !ok1 {
			continue
		}
		color, _ := s.linkColor(l)
		dash := ""
		if l.Kind == LinkFork {
			dash = fmt.Sprintf(` stroke-dasharray="%.1f %.1f"`, x.k*1.5, x.k*1.5)
		}
		fmt.Fprintf(&x.b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f"%s/>`+"\n",
			from.X*x.k, from.Y*x.k, to.X*x.k, to.Y*x.k, color, x.k*0.35, dash)
	}
	for _, env := range s.Graph.Envs {
		p, ok := s.Pos[env.ID]
		if !ok {
			continue
		}
		_, color := s.stationGlyph(env)
		if env.Unloaded || env.Restricted {
			x.circle(p, 1.5, colorFaint, 0.4, "")
		} else {
			x.disc(p, 1.8, color)
		}
		if env.ID == s.Selected {
			x.circle(p, 4, colorTeal, 0.4, "")
		}
		x.label(env, p, 6, 1, "start", stationBadges(env))
	}
}

// ExportName is the file an export writes, unique to the minute.
func ExportName(org string, at time.Time) string {
	safe := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, org)
	return fmt.Sprintf("constellation-%s-%s.svg", safe, at.Format("20060102-1504"))
}

type exportedMsg struct {
	path string
	err  error
}

func (m Model) exportMap() tea.Cmd {
	if m.graph == nil || m.cfg.WriteFile == nil {
		return nil
	}
	w, h := m.canvasSize()
	now := m.cfg.Now()
	scene := Scene{
		Graph: m.graph, Pos: m.pos, Selected: m.selected, Blast: m.blast(), Theme: m.cfg.Theme,
		ShowApps: m.showApps, GroupByProject: m.grouped, ShowCost: m.showCost, Filter: m.filter, Now: now,
	}
	title := m.cfg.Org
	if env := m.graph.Env(m.focus); env != nil {
		title += " · " + env.ProjectName + " / " + env.Name
	}
	title += " · " + now.Format("2006-01-02 15:04 MST")
	path, write := ExportName(m.cfg.Org, now), m.cfg.WriteFile
	data := []byte(ExportSVG(scene, w, h, title))
	return func() tea.Msg { return exportedMsg{path: path, err: write(path, data)} }
}

func (m Model) applyExported(msg exportedMsg) Model {
	m.toast, m.toastAt = "▣ saved the map to "+msg.path, m.frame
	if msg.err != nil {
		m.toast = "▣ could not save the map: " + clean(msg.err.Error())
	}
	return m
}
