package dashboard

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Scene is what the constellation shows on one frame.
type Scene struct {
	Graph          *Graph
	Pos            map[string]Point
	Selected       string
	Blast          *Blast
	ShowApps       bool
	GroupByProject bool
	ShowCost       bool
	Filter         string
	Frame          int
	// Intro runs from 0 to 1 while the constellation unfolds from the center;
	// zero or one shows it settled.
	Intro float64
	// Flashes holds, for each environment with a fresh event, how many frames
	// ago the event arrived and what it was.
	Flashes map[string]Flash
	// Target is the environment an incident just took the view to, and
	// TargetAge how many frames ago.
	Target    string
	TargetAge int
	// Now is when the frame is drawn, for traffic that has gone stale.
	Now time.Time
	// Theme is how the map is drawn; empty is the constellation.
	Theme string
	// Blocks draws dots as solid blocks rather than braille.
	Blocks bool
}

// Flash marks an environment where something just happened.
type Flash struct {
	Age   int
	Label string
}

// FlashFrames is how long a flash stays on screen.
const FlashFrames = 30

const (
	layerStars = iota
	layerLinks
	layerHotLinks
	layerParticles
	layerSatellites
	layerNodes
	layerHalo
)

const (
	nodeRadius      = 4.5
	satelliteRadius = 9.0
)

// Render draws the scene into a w×h block of cells.
func (s Scene) Render(w, h int) string {
	c := borrowCanvas(w, h)
	defer c.release()
	c.Blocks = s.Blocks
	if s.Theme == ThemePlain {
		s.renderPlain(c)
		s.drawFlashes(c)
		return c.Render()
	}
	s.Pos = s.unfold(c)
	s.drawStars(c)
	switch s.Theme {
	case ThemeMetro:
		s.renderMetro(c)
		s.drawFlashes(c)
		s.drawLockOn(c)
		return c.Render()
	case ThemeRadar:
		s.renderRadar(c)
		s.drawFlashes(c)
		s.drawLockOn(c)
		return c.Render()
	}
	s.drawLinks(c)
	if s.settled() {
		s.drawParticles(c)
	}
	if s.ShowApps {
		s.drawSatellites(c)
	}
	for _, env := range s.Graph.Envs {
		s.drawNode(c, env)
	}
	for _, env := range s.Graph.Envs {
		s.drawLabel(c, env)
	}
	s.drawTrafficLabels(c)
	s.drawFlashes(c)
	s.drawLockOn(c)
	return c.Render()
}

func (s Scene) settled() bool { return s.Intro <= 0 || s.Intro >= 1 }

func (s Scene) unfold(c *Canvas) map[string]Point {
	if s.settled() {
		return s.Pos
	}
	u := 1 - s.Intro
	t := 1 - u*u*u
	center := Point{X: float64(c.DotW()) / 2, Y: float64(c.DotH()) / 2}
	out := make(map[string]Point, len(s.Pos))
	for id, p := range s.Pos {
		out[id] = Point{X: center.X + (p.X-center.X)*t, Y: center.Y + (p.Y-center.Y)*t}
	}
	return out
}

// Matches reports whether env matches the search filter.
func (s Scene) Matches(env *EnvNode) bool {
	if s.Filter == "" {
		return true
	}
	q := strings.ToLower(s.Filter)
	for _, field := range []string{env.ID, env.Name, env.ProjectName, env.ProjectID} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	for _, inst := range env.Instances {
		if strings.Contains(strings.ToLower(inst.ID), q) {
			return true
		}
	}
	return false
}

func hash01(x, y, salt int) float64 {
	n := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(salt*83492791) //nolint:gosec // a hash, overflow is the point
	n ^= n >> 13
	n *= 0x5bd1e995
	n ^= n >> 15
	return float64(n%10000) / 10000
}

func (s Scene) drawStars(c *Canvas) { drawBackground(c, s.Frame) }

func (s Scene) linkColor(l Link) (lipgloss.Color, int) {
	if s.Blast != nil {
		hopFrom, okFrom := s.Blast.Reaches(l.From)
		hopTo, okTo := s.Blast.Reaches(l.To)
		if okFrom && okTo && l.Kind == LinkDependency && hopTo > hopFrom {
			return hopColor(hopTo), layerHotLinks
		}
	}
	if s.Selected != "" && (l.From == s.Selected || l.To == s.Selected) {
		return colorTeal, layerHotLinks
	}
	from, to := s.Graph.Env(l.From), s.Graph.Env(l.To)
	if from != nil && to != nil && !s.Matches(from) && !s.Matches(to) {
		return colorLinkDim, layerLinks
	}
	if l.Kind == LinkFork {
		return colorTealDim, layerLinks
	}
	return colorLink, layerLinks
}

func (s Scene) curve(l Link) (Point, Point, Point, bool) {
	p0, ok0 := s.Pos[l.From]
	p2, ok2 := s.Pos[l.To]
	return p0, Bend(p0, p2, 0.15), p2, ok0 && ok2
}

func (s Scene) drawLinks(c *Canvas) {
	until := 0.0
	if !s.settled() {
		until = (s.Intro - 0.45) / 0.55
		if until <= 0 {
			return
		}
	}
	for _, l := range s.Graph.EnvLinks {
		p0, p1, p2, ok := s.curve(l)
		if !ok {
			continue
		}
		color, layer := s.linkColor(l)
		st := Stroke{Color: color, Layer: layer, Gap: nodeRadius + 1.5, Until: until}
		if l.Kind == LinkFork {
			st.Dash = 2
		}
		c.Curve(p0, p1, p2, st)
	}
}

func (s Scene) drawParticles(c *Canvas) {
	for i, l := range s.Graph.EnvLinks {
		if l.Kind != LinkDependency {
			continue
		}
		p0, p1, p2, ok := s.curve(l)
		if !ok {
			continue
		}
		color, _ := s.linkColor(l)
		if color == colorLink || color == colorLinkDim {
			color = colorTeal
		}
		sparks, speed, errs := sparkFlow(l, s.Graph.Traffic(l))
		speed /= math.Max(p0.Dist(p2), 1)
		for k := range sparks {
			t := math.Mod(float64(s.Frame)*speed+float64(k)/float64(sparks)+float64(i)*0.137, 1)
			spark := color
			if k < errs {
				spark = errorColor(s.Graph.Traffic(l).Errors)
			}
			c.Disc(Bezier(p0, p1, p2, t), 0.6, spark, layerParticles)
			if t > 0.03 {
				tail := Bezier(p0, p1, p2, t-0.03)
				c.Dot(int(math.Round(tail.X)), int(math.Round(tail.Y)), spark, layerParticles)
			}
		}
	}
}

// sparkFlow turns a link's traffic into how many sparks run along it, how
// fast, and how many of them are red for errors. Without traffic, a link with
// more resources gets more sparks.
func sparkFlow(l Link, tr *Traffic) (int, float64, int) {
	if tr == nil {
		return min(l.Count, 3), 1.2, 0
	}
	if tr.Rate <= 0 {
		return 0, 0, 0
	}
	weight := math.Log10(tr.Rate + 1)
	sparks := min(max(1+int(weight*1.5), 1), 6)
	errs := 0
	if tr.Errors > 0 {
		errs = max(1, min(sparks/2, int(math.Log10(float64(tr.Errors)+1)*2)))
	}
	return sparks, 1.2 * (1 + weight/3), errs
}

// errorColor follows the Massdriver UI: a few errors are amber, ten or more
// are red.
func errorColor(errors int) lipgloss.Color {
	if errors >= manyErrors {
		return colorRed
	}
	return colorOrange
}

const manyErrors = 10

func (s Scene) drawTrafficLabels(c *Canvas) {
	if !s.settled() {
		return
	}
	for _, l := range s.Graph.EnvLinks {
		tr := s.Graph.Traffic(l)
		p0, p1, p2, ok := s.curve(l)
		if tr == nil || !ok {
			continue
		}
		mid := Bezier(p0, p1, p2, 0.5)
		label := Rate(tr.Rate) + "/s" + latencyText(tr)
		style := lipgloss.NewStyle().Foreground(colorTealDim)
		if tr.Errors > 0 {
			label += fmt.Sprintf(" ✕%d", tr.Errors)
			style = lipgloss.NewStyle().Foreground(errorColor(tr.Errors))
		}
		if tr.Stale(s.Now) {
			label += " stale"
			style = styleFaint
		}
		w := runewidth.StringWidth(label)
		if x, y := int(mid.X)/2-w/2, int(mid.Y)/4; c.Free(x-1, y, w+2) {
			c.Text(x, y, label, style)
		}
	}
}

// Rate formats a count compactly, like 950, 1.2k, or 3.4M.
func Rate(n float64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", n/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", n/1000)
	case n >= 10 || n == math.Trunc(n):
		return fmt.Sprintf("%.0f", n)
	default:
		return fmt.Sprintf("%.1f", n)
	}
}

func (s Scene) drawSatellites(c *Canvas) {
	for _, env := range s.Graph.Envs {
		center, ok := s.Pos[env.ID]
		if !ok || len(env.Instances) == 0 {
			continue
		}
		n := len(env.Instances)
		r := s.radius(env) + satelliteRadius - nodeRadius
		for i, inst := range env.Instances {
			angle := 2*math.Pi*float64(i)/float64(n) - math.Pi/2 + 0.3
			p := Point{X: center.X + r*1.4*math.Cos(angle), Y: center.Y + r*math.Sin(angle)}
			color := s.instanceColor(env, inst)
			if inst.IsApp() {
				c.Disc(p, 1, color, layerSatellites)
			} else {
				c.Dot(int(math.Round(p.X)), int(math.Round(p.Y)), color, layerSatellites)
			}
		}
	}
}

func (s Scene) instanceColor(env *EnvNode, inst *InstNode) lipgloss.Color {
	if hop, ok := s.Blast.Reaches(inst.ID); ok && hop > 0 {
		return hopColor(hop)
	}
	switch {
	case inst.Firing > 0 && s.Frame%4 < 2:
		return colorRed
	case inst.Firing > 0:
		return colorRedDim
	case inst.Deploying() && s.Frame%6 < 3:
		return colorHighlight
	case !s.Matches(env):
		return colorFaint
	case healthColor(inst.Health) == colorRed && s.Frame%6 < 3:
		return colorRedDim
	case inst.Health != "":
		return healthColor(inst.Health)
	}
	return statusColor(inst.Status)
}

func (s Scene) ringColors(env *EnvNode) func(float64) lipgloss.Color {
	if env.Restricted || !s.Matches(env) || len(env.Instances) == 0 {
		return func(float64) lipgloss.Color { return colorFaint }
	}
	total := float64(len(env.Instances))
	return func(frac float64) lipgloss.Color {
		acc := 0.0
		for _, status := range Statuses {
			acc += float64(env.StatusCount(status)) / total
			if frac < acc {
				return statusColor(status)
			}
		}
		return colorGray
	}
}

func (s Scene) radius(env *EnvNode) float64 {
	if !s.ShowCost {
		return nodeRadius
	}
	peak := 0.0
	for _, e := range s.Graph.Envs {
		peak = math.Max(peak, e.Cost)
	}
	if peak == 0 {
		return nodeRadius
	}
	return nodeRadius * (0.7 + 1.1*math.Sqrt(env.Cost/peak))
}

func (s Scene) drawNode(c *Canvas, env *EnvNode) {
	center, ok := s.Pos[env.ID]
	if !ok {
		return
	}
	r := s.radius(env)
	c.Ring(center, r, s.ringColors(env), layerNodes)
	if env.FiringCount() > 0 {
		s.drawAlarmPulse(c, center, r)
	}
	if len(env.Deploying()) > 0 {
		s.drawDeploySpinner(c, center, r)
	}
	if hop, ok := s.Blast.Reaches(env.ID); ok {
		c.Ring(center, r+2.5, func(float64) lipgloss.Color { return hopColor(hop) }, layerHalo)
	}
	if env.ID == s.Selected {
		c.Ring(center, r+4, func(f float64) lipgloss.Color {
			if math.Mod(f*8+float64(s.Frame)*0.08, 2) < 1 {
				return colorHighlight
			}
			return colorTeal
		}, layerHalo)
	}
}

func (s Scene) drawAlarmPulse(c *Canvas, center Point, r float64) {
	c.Disc(center, 1.5, colorRed, layerNodes)
	phase := s.Frame % 12
	color := colorRed
	if phase > 6 {
		color = colorRedDim
	}
	c.Ring(center, r+2.5+float64(phase)*0.8, func(float64) lipgloss.Color { return color }, layerHalo)
}

func (s Scene) drawDeploySpinner(c *Canvas, center Point, r float64) {
	head := math.Mod(float64(s.Frame)*0.045, 1)
	c.Ring(center, r+1.2, func(f float64) lipgloss.Color {
		behind := math.Mod(head-f+1, 1)
		switch {
		case behind < 0.06:
			return colorHighlight
		case behind < 0.3:
			return colorBlue
		default:
			return ""
		}
	}, layerHalo)
}

func (s Scene) labelStyles(env *EnvNode) (lipgloss.Style, lipgloss.Style) {
	name := styleTitle
	sub := styleMuted
	if s.GroupByProject {
		sub = lipgloss.NewStyle().Foreground(projectColor(s.Graph, env.ProjectID))
	}
	switch {
	case env.ID == s.Selected:
		name = styleSelected
	case !s.Matches(env) || env.Restricted || env.Unloaded:
		name, sub = styleFaint, styleFaint
	case env.FiringCount() > 0:
		name = styleBoldRed
	}
	return name, sub
}

func (s Scene) subLabel(env *EnvNode) string {
	switch {
	case env.Restricted:
		return "no access"
	case env.Unloaded:
		return "🔭 out of view"
	case s.ShowCost && env.Currency != "":
		return env.ProjectName + " · " + Money(env.Cost, env.Currency) + "/mo"
	default:
		return env.ProjectName
	}
}

func (s Scene) drawLabel(c *Canvas, env *EnvNode) {
	center, ok := s.Pos[env.ID]
	if !ok {
		return
	}
	cx, cy := int(center.X)/2, int(center.Y)/4
	nameStyle, subStyle := s.labelStyles(env)
	name := env.Name
	if env.ID == s.Selected {
		name = " " + name + " "
	}
	c.TextCentered(cx, cy+2, name, nameStyle)
	c.TextCentered(cx, cy+3, s.subLabel(env), subStyle)
	if n := env.FiringCount(); n > 0 {
		c.Text(cx+3, cy-1, fmt.Sprintf("▲%d", n), styleBoldRed)
	}
	if n := env.Upgrades(); n > 0 {
		badge := fmt.Sprintf("↑%d", n)
		c.Text(cx-3-len([]rune(badge)), cy-1, badge, styleUpgrade)
	}
	if env.Banner != nil {
		c.Text(cx-runewidth.StringWidth(name)/2-2, cy+2, "⚑", lipgloss.NewStyle().Bold(true).Foreground(levelColor(env.Banner.Level)))
	}
}

func (s Scene) drawFlashes(c *Canvas) {
	ids := make([]string, 0, len(s.Flashes))
	for id := range s.Flashes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		f := s.Flashes[id]
		center, ok := s.Pos[id]
		if !ok || f.Age >= FlashFrames {
			continue
		}
		color := colorHighlight
		if f.Age > FlashFrames/3 {
			color = colorTeal
		}
		if f.Age > 2*FlashFrames/3 {
			color = colorTealDim
		}
		c.Ring(center, nodeRadius+float64(f.Age)*0.7, func(float64) lipgloss.Color { return color }, layerHalo)
		cx, cy := int(center.X)/2, int(center.Y)/4
		c.TextCentered(cx, cy-2-f.Age/10, f.Label, lipgloss.NewStyle().Foreground(color))
	}
}
