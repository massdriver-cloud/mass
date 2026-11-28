package dashboard

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Themes: how the map is drawn.
const (
	// ThemeConstellation scatters environments like stars, pulled together by
	// the resources they share.
	ThemeConstellation = "constellation"
	// ThemeMetro draws environments as stations on transit lines: one line a
	// project, producers to the left of their consumers.
	ThemeMetro = "metro"
	// ThemeRadar puts the focus at the center of a radar screen, with each
	// environment on the ring as many hops out as it is.
	ThemeRadar = "radar"

	// DefaultTheme is the theme the dashboard starts in.
	DefaultTheme = ThemeConstellation
)

// Themes lists the theme names, the default first.
func Themes() []string { return []string{ThemeConstellation, ThemeMetro, ThemeRadar} }

// NextTheme is the theme after name, wrapping around.
func NextTheme(name string) string {
	all := Themes()
	return all[(slices.Index(all, name)+1)%len(all)]
}

// LayoutTheme places every environment for a theme, in dots.
func LayoutTheme(theme string, g *Graph, w, h int, opts LayoutOptions) map[string]Point {
	pos, _ := layoutTheme(theme, g, w, h, opts)
	return pos
}

// layoutTheme is LayoutTheme that also returns the constellation's settled
// positions, to seed the next layout with; other themes return none.
func layoutTheme(theme string, g *Graph, w, h int, opts LayoutOptions) (map[string]Point, map[string]Point) {
	switch theme {
	case ThemeMetro:
		return MetroLayout(g, w, h), nil
	case ThemeRadar:
		return RadarLayout(g, w, h, opts.Center), nil
	}
	return layout(g, w, h, opts)
}

// hops is how many links apart each environment is from center, following
// links either way. Environments center cannot reach are left out.
func hops(g *Graph, center string) map[string]int {
	adjacent := map[string][]string{}
	for _, l := range g.EnvLinks {
		adjacent[l.From] = append(adjacent[l.From], l.To)
		adjacent[l.To] = append(adjacent[l.To], l.From)
	}
	out := map[string]int{center: 0}
	frontier := []string{center}
	for hop := 1; len(frontier) > 0; hop++ {
		var next []string
		for _, id := range frontier {
			for _, n := range adjacent[id] {
				if _, ok := out[n]; !ok {
					out[n] = hop
					next = append(next, n)
				}
			}
		}
		frontier = next
	}
	return out
}

// RadarLayout puts center in the middle and each environment on the ring as
// many hops out as it is; the ones center cannot reach sit on the edge.
func RadarLayout(g *Graph, w, h int, center string) map[string]Point {
	out := map[string]Point{}
	if len(g.Envs) == 0 {
		return out
	}
	if g.Env(center) == nil {
		center = envOrder(g)[0].ID
	}
	dist := hops(g, center)
	far := 0
	for _, d := range dist {
		far = max(far, d)
	}
	rings := map[int][]string{}
	for _, env := range envOrder(g) {
		d, ok := dist[env.ID]
		if !ok {
			d = far + 1
		}
		rings[d] = append(rings[d], env.ID)
	}
	cx, cy := float64(w), float64(h*2)
	rx, ry := math.Max(float64(w)-24, 8), math.Max(float64(h*2)-8, 4)
	outer := far + 1
	if len(rings[outer]) == 0 {
		outer = far
	}
	for d, ids := range rings {
		frac := 0.0
		if outer > 0 {
			frac = float64(d) / float64(outer)
		}
		for i, id := range ids {
			angle := 2*math.Pi*float64(i)/float64(len(ids)) + float64(d)*0.7 - math.Pi/2
			out[id] = Point{X: cx + rx*frac*math.Cos(angle), Y: cy + ry*frac*math.Sin(angle)}
		}
	}
	return out
}

const (
	metroRowHeight = 3
	metroMinColumn = 18
)

// MetroLayout lays environments out like a transit map: a row band for each
// project's line, and a column for each layer of producers, so resources
// flow left to right.
func MetroLayout(g *Graph, w, h int) map[string]Point {
	layers := g.Layers()
	cols := 1
	for _, l := range layers {
		cols = max(cols, l+1)
	}
	byProject := map[string][]*EnvNode{}
	var projects []string
	for _, env := range envOrder(g) {
		if _, ok := byProject[env.ProjectID]; !ok {
			projects = append(projects, env.ProjectID)
		}
		byProject[env.ProjectID] = append(byProject[env.ProjectID], env)
	}
	colW := max((w-4)/cols, metroMinColumn)
	out := map[string]Point{}
	row := 0
	for _, project := range projects {
		used := map[int]int{}
		band := 1
		envs := byProject[project]
		slices.SortStableFunc(envs, func(a, b *EnvNode) int { return cmp.Compare(layers[a.ID], layers[b.ID]) })
		for _, env := range envs {
			col := layers[env.ID]
			sub := used[col]
			used[col]++
			band = max(band, used[col])
			x := 2 + col*colW
			y := 1 + (row+sub)*metroRowHeight
			out[env.ID] = Point{X: float64(x*2 + 1), Y: float64(y*4 + 2)}
		}
		row += band
	}
	if rows := row * metroRowHeight; rows > h-2 && rows > 0 {
		squeeze := float64(h-2) / float64(rows)
		for id, p := range out {
			out[id] = Point{X: p.X, Y: math.Max(p.Y*squeeze, 2)}
		}
	}
	return out
}

func cellOf(p Point) (int, int) { return int(p.X) / 2, int(p.Y) / 4 }

// metroRoute is the cells of a line from one station to another: across,
// down or up at the halfway column, then across again.
func metroRoute(from, to Point, lane int) [][2]int {
	x0, y0 := cellOf(from)
	x1, y1 := cellOf(to)
	mid := (x0+x1)/2 + lane%3 - 1
	if x1 <= x0+2 {
		mid = x0 + 2
	}
	var cells [][2]int
	stepTo := func(x, y, tx, ty int) (int, int) {
		for x != tx || y != ty {
			switch {
			case x < tx:
				x++
			case x > tx:
				x--
			case y < ty:
				y++
			default:
				y--
			}
			cells = append(cells, [2]int{x, y})
		}
		return x, y
	}
	x, y := stepTo(x0, y0, mid, y0)
	x, y = stepTo(x, y, mid, y1)
	stepTo(x, y, x1, y1)
	if len(cells) > 0 {
		cells = cells[:len(cells)-1]
	}
	return cells
}

// trackGlyph is the box-drawing piece for a cell of a route, from the
// directions it comes from and goes to.
func trackGlyph(prev, cur, next [2]int, dashed bool) string {
	in := [2]int{cur[0] - prev[0], cur[1] - prev[1]}
	out := [2]int{next[0] - cur[0], next[1] - cur[1]}
	horizontal, vertical := "━", "┃"
	if dashed {
		horizontal, vertical = "┅", "┇"
	}
	switch {
	case in == out && in[1] == 0:
		return horizontal
	case in == out:
		return vertical
	case (in == [2]int{1, 0} && out == [2]int{0, 1}) || (in == [2]int{0, -1} && out == [2]int{-1, 0}):
		return "┓"
	case (in == [2]int{1, 0} && out == [2]int{0, -1}) || (in == [2]int{0, 1} && out == [2]int{-1, 0}):
		return "┛"
	case (in == [2]int{0, 1} && out == [2]int{1, 0}) || (in == [2]int{-1, 0} && out == [2]int{0, -1}):
		return "┗"
	default:
		return "┏"
	}
}

func (s Scene) renderMetro(c *Canvas) {
	routes := make([][][2]int, len(s.Graph.EnvLinks))
	for i, l := range s.Graph.EnvLinks {
		from, ok0 := s.Pos[l.From]
		to, ok1 := s.Pos[l.To]
		if !ok0 || !ok1 {
			continue
		}
		routes[i] = metroRoute(from, to, i)
		color, _ := s.linkColor(l)
		if color == colorLink || color == colorTealDim {
			color = projectColor(s.Graph, s.Graph.Env(l.From).ProjectID)
		}
		style := c.Style(lipgloss.NewStyle().Foreground(color))
		fx, fy := cellOf(from)
		tx, ty := cellOf(to)
		path := append(append([][2]int{{fx, fy}}, routes[i]...), [2]int{tx, ty})
		for j := 1; j+1 < len(path); j++ {
			c.Put(path[j][0], path[j][1], trackGlyph(path[j-1], path[j], path[j+1], l.Kind == LinkFork), style)
		}
	}
	if s.settled() {
		s.metroSparks(c, routes)
	}
	for _, env := range s.Graph.Envs {
		s.metroStation(c, env)
	}
}

func (s Scene) metroSparks(c *Canvas, routes [][][2]int) {
	for i, l := range s.Graph.EnvLinks {
		route := routes[i]
		if l.Kind != LinkDependency || len(route) == 0 {
			continue
		}
		sparks, speed, errs := sparkFlow(l, s.Graph.Traffic(l))
		for k := range sparks {
			pos := int(float64(s.Frame)*speed*0.6+float64(k*len(route))/float64(max(sparks, 1))+float64(i*7)) % len(route)
			color := colorHighlight
			if k < errs {
				color = errorColor(s.Graph.Traffic(l).Errors)
			}
			c.Text(route[pos][0], route[pos][1], "●", lipgloss.NewStyle().Foreground(color))
		}
	}
}

var spinner = []string{"◐", "◓", "◑", "◒"}

func (s Scene) stationGlyph(env *EnvNode) (string, lipgloss.Color) {
	switch {
	case env.Unloaded:
		return "○", colorFaint
	case env.Restricted:
		return "◌", colorFaint
	case env.FiringCount() > 0 && s.Frame%8 < 4:
		return "◉", colorRed
	case env.FiringCount() > 0:
		return "○", colorRedDim
	case len(env.Deploying()) > 0:
		return spinner[(s.Frame/3)%len(spinner)], colorBlue
	}
	if hop, ok := s.Blast.Reaches(env.ID); ok {
		return "◉", hopColor(hop)
	}
	return "◉", statusColor(env.Health())
}

func (s Scene) metroStation(c *Canvas, env *EnvNode) {
	p, ok := s.Pos[env.ID]
	if !ok {
		return
	}
	x, y := cellOf(p)
	glyph, color := s.stationGlyph(env)
	c.Text(x, y, glyph, lipgloss.NewStyle().Bold(true).Foreground(color))
	nameStyle, subStyle := s.labelStyles(env)
	name := env.Name
	if env.ID == s.Selected {
		name = " " + name + " "
	}
	c.Text(x+2, y, name, nameStyle)
	label := s.subLabel(env)
	if env.FiringCount() > 0 {
		label += fmt.Sprintf(" ▲%d", env.FiringCount())
	}
	if n := env.Upgrades(); n > 0 {
		label += fmt.Sprintf(" ↑%d", n)
	}
	if env.Banner != nil {
		label += " ⚑"
	}
	c.Text(x+2, y+1, label, subStyle)
}

const radarSweepSpeed = 0.07

func (s Scene) renderRadar(c *Canvas) {
	cx, cy := float64(c.DotW())/2, float64(c.DotH())/2
	s.radarScope(c, cx, cy)
	for _, l := range s.Graph.EnvLinks {
		from, ok0 := s.Pos[l.From]
		to, ok1 := s.Pos[l.To]
		if !ok0 || !ok1 {
			continue
		}
		color, layer := s.linkColor(l)
		dash := 0
		if l.Kind == LinkFork {
			dash = 2
		}
		c.Line(from.X, from.Y, to.X, to.Y, color, layer, dash)
	}
	sweep := math.Mod(float64(s.Frame)*radarSweepSpeed, 2*math.Pi)
	for _, env := range s.Graph.Envs {
		s.radarBlip(c, env, cx, cy, sweep)
	}
}

func (s Scene) radarScope(c *Canvas, cx, cy float64) {
	rx, ry := float64(c.DotW())/2-24, float64(c.DotH())/2-8
	for _, frac := range []float64{1.0 / 3, 2.0 / 3, 1} {
		steps := int(2 * math.Pi * math.Max(rx, ry) * frac)
		for i := range steps {
			if i%3 != 0 {
				continue
			}
			a := 2 * math.Pi * float64(i) / float64(steps)
			c.Dot(int(cx+rx*frac*math.Cos(a)), int(cy+ry*frac*math.Sin(a)), colorStarLit, layerStars)
		}
	}
	c.Line(cx-rx, cy, cx+rx, cy, colorStar, layerStars, 3)
	c.Line(cx, cy-ry, cx, cy+ry, colorStar, layerStars, 3)
	sweep := math.Mod(float64(s.Frame)*radarSweepSpeed, 2*math.Pi)
	trail := []lipgloss.Color{colorHighlight, colorTeal, colorTeal, colorTealDim, colorTealDim, colorStarLit}
	for k, color := range trail {
		a := sweep - float64(k)*0.05
		c.Line(cx, cy, cx+rx*math.Cos(a), cy+ry*math.Sin(a), color, layerLinks-k%2, 0)
	}
}

func (s Scene) radarBlip(c *Canvas, env *EnvNode, cx, cy, sweep float64) {
	p, ok := s.Pos[env.ID]
	if !ok {
		return
	}
	angle := math.Atan2((p.Y-cy)/math.Max(float64(c.DotH())/2-8, 1), (p.X-cx)/math.Max(float64(c.DotW())/2-24, 1))
	behind := math.Mod(sweep-angle+4*math.Pi, 2*math.Pi)
	_, color := s.stationGlyph(env)
	r := 1.5
	if behind < 0.5 {
		color = colorHighlight
		r = 2
		c.Ring(p, 3+behind*8, func(float64) lipgloss.Color { return colorTeal }, layerHalo)
	}
	if env.Unloaded || env.Restricted {
		c.Ring(p, 1.5, func(float64) lipgloss.Color { return colorFaint }, layerNodes)
	} else {
		c.Disc(p, r, color, layerNodes)
	}
	if env.ID == s.Selected {
		c.Ring(p, 4, func(f float64) lipgloss.Color {
			if math.Mod(f*8+float64(s.Frame)*0.08, 2) < 1 {
				return colorHighlight
			}
			return colorTeal
		}, layerHalo)
	}
	x, y := cellOf(p)
	nameStyle, subStyle := s.labelStyles(env)
	name := env.Name
	if env.FiringCount() > 0 {
		name = fmt.Sprintf("▲%d ", env.FiringCount()) + name
	}
	c.Text(x+3, y, name, nameStyle)
	sub := s.subLabel(env)
	if env.Banner != nil {
		sub += " ⚑"
	}
	c.Text(x+3, y+1, runewidth.Truncate(sub, 28, "…"), subStyle)
}
