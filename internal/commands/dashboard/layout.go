package dashboard

import (
	"cmp"
	"hash/fnv"
	"math"
	"slices"

	"github.com/mattn/go-runewidth"
)

// LayoutOptions shape how the constellation is placed.
type LayoutOptions struct {
	// GroupByProject pulls environments of the same project together.
	GroupByProject bool
	// ProducersOnTop puts each environment above the environments that use its
	// resources.
	ProducersOnTop bool
	// Center is the environment the layout holds in the middle.
	Center string
	// Seed is an earlier layout's settled positions. Starting from it moves
	// what is already on screen as little as the change allows, so a new
	// focus pans the map rather than reshuffling it.
	Seed map[string]Point
}

// Layout places every environment in a w×h cell box, returning each one's
// node center in dots. The same graph and options always give the same layout.
func Layout(g *Graph, w, h int, opts LayoutOptions) map[string]Point {
	pos, _ := layout(g, w, h, opts)
	return pos
}

// layout is Layout that also returns the settled positions, to seed the
// next layout with.
func layout(g *Graph, w, h int, opts LayoutOptions) (map[string]Point, map[string]Point) {
	if len(g.Envs) == 0 {
		return map[string]Point{}, nil
	}
	pos := settle(g, opts)
	half := labelHalfWidths(g)
	var cells map[string]Point
	if opts.ProducersOnTop {
		cells = stackLayers(g, pos, half, w, h)
	} else {
		cells = fit(pos, w, h, opts.Center)
	}
	separate(g, cells, half, w, h)
	out := make(map[string]Point, len(cells))
	for id, p := range cells {
		out[id] = Point{X: p.X*2 + 1, Y: p.Y*4 + 2}
	}
	return out, pos
}

func envOrder(g *Graph) []*EnvNode {
	envs := slices.Clone(g.Envs)
	slices.SortFunc(envs, func(a, b *EnvNode) int {
		return cmp.Or(cmp.Compare(a.ProjectID, b.ProjectID), cmp.Compare(a.ID, b.ID))
	})
	return envs
}

func jitter(id string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return float64(h.Sum32()%1000)/1000 - 0.5
}

const (
	layoutIterations = 400
	// A seeded layout starts near its answer, so it runs fewer, gentler
	// steps, which also keeps it from drifting from the seed.
	seededIterations  = 120
	seededTemperature = 0.02
)

func settle(g *Graph, opts LayoutOptions) map[string]Point {
	envs := envOrder(g)
	index := make(map[string]int, len(envs))
	for i, env := range envs {
		index[env.ID] = i
	}
	links := make([][2]int, 0, len(g.EnvLinks))
	for _, l := range g.EnvLinks {
		a, okA := index[l.From]
		b, okB := index[l.To]
		if okA && okB {
			links = append(links, [2]int{a, b})
		}
	}
	pos, temp, iterations := seeded(envs, links, opts)
	if pos == nil {
		pos, temp, iterations = make([]Point, len(envs)), 0.1, layoutIterations
		for i, env := range envs {
			angle := 2 * math.Pi * float64(i) / float64(len(envs))
			r := 0.35 + 0.05*jitter(env.ID)
			pos[i] = Point{X: r * math.Cos(angle), Y: r * math.Sin(angle)}
		}
	}
	center, pinned := index[opts.Center]
	k := math.Sqrt(1.0 / float64(len(envs)))
	disp := make([]Point, len(envs))
	for range iterations {
		clear(disp)
		repel(pos, disp, k)
		attract(links, pos, disp, k)
		if opts.GroupByProject {
			cluster(envs, pos, disp, k)
		}
		for i, p := range pos {
			if pinned && i == center {
				pos[i] = Point{}
				continue
			}
			d := disp[i]
			dist := math.Hypot(p.X, p.Y)
			d.X -= p.X * dist * 4 / k
			d.Y -= p.Y * dist * 4 / k
			length := math.Max(math.Hypot(d.X, d.Y), 1e-9)
			step := math.Min(length, temp)
			pos[i] = Point{X: p.X + d.X/length*step, Y: p.Y + d.Y/length*step}
		}
		temp = math.Max(temp*0.985, 0.002)
	}
	out := make(map[string]Point, len(envs))
	for i, env := range envs {
		out[env.ID] = pos[i]
	}
	return out
}

// seeded starts from the seed shifted so the center sits at the origin. An
// environment new to the layout starts beside the neighbors it already has,
// or out on the edge if it has none. A seed without the center is no help,
// so settle starts afresh.
func seeded(envs []*EnvNode, links [][2]int, opts LayoutOptions) ([]Point, float64, int) {
	origin, ok := opts.Seed[opts.Center]
	if !ok {
		return nil, 0, 0
	}
	pos := make([]Point, len(envs))
	placed := make([]bool, len(envs))
	for i, env := range envs {
		if p, ok := opts.Seed[env.ID]; ok {
			pos[i], placed[i] = Point{X: p.X - origin.X, Y: p.Y - origin.Y}, true
		}
	}
	for i, env := range envs {
		if placed[i] {
			continue
		}
		var sum Point
		n := 0
		for _, l := range links {
			other := l[0]
			if other == i {
				other = l[1]
			} else if l[1] != i {
				continue
			}
			if placed[other] {
				sum, n = Point{X: sum.X + pos[other].X, Y: sum.Y + pos[other].Y}, n+1
			}
		}
		angle := 2 * math.Pi * (jitter(env.ID) + 0.5)
		if n == 0 {
			pos[i] = Point{X: 0.5 * math.Cos(angle), Y: 0.5 * math.Sin(angle)}
		} else {
			pos[i] = Point{X: sum.X/float64(n) + 0.05*math.Cos(angle), Y: sum.Y/float64(n) + 0.05*math.Sin(angle)}
		}
		placed[i] = true
	}
	return pos, seededTemperature, seededIterations
}

func delta(a, b Point) (float64, float64, float64) {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx, dy, math.Max(math.Hypot(dx, dy), 1e-3)
}

func repel(pos, disp []Point, k float64) {
	for i := range pos {
		for j := i + 1; j < len(pos); j++ {
			dx, dy, d := delta(pos[i], pos[j])
			f := k * k / d / d
			disp[i].X, disp[i].Y = disp[i].X+dx*f, disp[i].Y+dy*f
			disp[j].X, disp[j].Y = disp[j].X-dx*f, disp[j].Y-dy*f
		}
	}
}

func attract(links [][2]int, pos, disp []Point, k float64) {
	for _, l := range links {
		a, b := l[0], l[1]
		dx, dy, d := delta(pos[a], pos[b])
		f := d / k
		disp[a].X, disp[a].Y = disp[a].X-dx*f, disp[a].Y-dy*f
		disp[b].X, disp[b].Y = disp[b].X+dx*f, disp[b].Y+dy*f
	}
}

func cluster(envs []*EnvNode, pos, disp []Point, k float64) {
	for i, a := range envs {
		for j := i + 1; j < len(envs); j++ {
			if a.ProjectID == "" || a.ProjectID != envs[j].ProjectID {
				continue
			}
			dx, dy, d := delta(pos[i], pos[j])
			f := 1.5 * d / k
			disp[i].X, disp[i].Y = disp[i].X-dx*f, disp[i].Y-dy*f
			disp[j].X, disp[j].Y = disp[j].X+dx*f, disp[j].Y+dy*f
		}
	}
}

func stackLayers(g *Graph, pos map[string]Point, half map[string]float64, w, h int) map[string]Point {
	layers := g.Layers()
	byLayer := map[int][]string{}
	deepest := 0
	for _, env := range envOrder(g) {
		byLayer[layers[env.ID]] = append(byLayer[layers[env.ID]], env.ID)
		deepest = max(deepest, layers[env.ID])
	}
	var rows [][]string
	for layer := 0; layer <= deepest; layer++ {
		ids := byLayer[layer]
		slices.SortStableFunc(ids, func(a, b string) int { return cmp.Compare(pos[a].X, pos[b].X) })
		rows = append(rows, wrapRow(ids, half, float64(w))...)
	}
	cells := make(map[string]Point, len(pos))
	for i, row := range rows {
		y := 1 + float64(max(h-5, 0))*(float64(i)+0.5)/float64(len(rows))
		for j, id := range row {
			cells[id] = Point{X: float64(w) * (float64(j) + 0.5) / float64(len(row)), Y: y}
		}
	}
	return cells
}

func wrapRow(ids []string, half map[string]float64, w float64) [][]string {
	var rows [][]string
	var cur []string
	used := 0.0
	for _, id := range ids {
		need := 2*half[id] + 2
		if len(cur) > 0 && used+need > w {
			rows = append(rows, cur)
			cur, used = nil, 0
		}
		cur = append(cur, id)
		used += need
	}
	if len(cur) > 0 {
		rows = append(rows, cur)
	}
	return rows
}

func labelHalfWidths(g *Graph) map[string]float64 {
	half := make(map[string]float64, len(g.Envs))
	for _, env := range g.Envs {
		half[env.ID] = float64(max(runewidth.StringWidth(env.Name), runewidth.StringWidth(env.ProjectName), 6))/2 + 1
	}
	return half
}

const labelRows = 5

func fit(pos map[string]Point, w, h int, center string) map[string]Point {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range pos {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	if c, ok := pos[center]; ok {
		rx := math.Max(math.Abs(minX-c.X), math.Abs(maxX-c.X))
		ry := math.Max(math.Abs(minY-c.Y), math.Abs(maxY-c.Y))
		minX, maxX, minY, maxY = c.X-rx, c.X+rx, c.Y-ry, c.Y+ry
	}
	padX, padTop, padBottom := 10.0, 2.0, 3.0
	spanX := math.Max(float64(w)-2*padX-1, 1)
	spanY := math.Max(float64(h)-padTop-padBottom-1, 1)
	out := make(map[string]Point, len(pos))
	for id, p := range pos {
		fx, fy := 0.5, 0.5
		if maxX > minX {
			fx = (p.X - minX) / (maxX - minX)
		}
		if maxY > minY {
			fy = (p.Y - minY) / (maxY - minY)
		}
		out[id] = Point{X: padX + fx*spanX, Y: padTop + fy*spanY}
	}
	return out
}

func separate(g *Graph, cells map[string]Point, half map[string]float64, w, h int) {
	envs := envOrder(g)
	for range 80 {
		moved := false
		for i, a := range envs {
			for _, b := range envs[i+1:] {
				if nudge(cells, a.ID, b.ID, half[a.ID]+half[b.ID], labelRows) {
					moved = true
				}
			}
		}
		clamp(cells, half, w, h)
		if !moved {
			return
		}
	}
}

func nudge(cells map[string]Point, a, b string, minDX, minDY float64) bool {
	pa, pb := cells[a], cells[b]
	dx, dy := pb.X-pa.X, pb.Y-pa.Y
	overX, overY := minDX-math.Abs(dx), minDY-math.Abs(dy)
	if overX <= 0 || overY <= 0 {
		return false
	}
	// Move along whichever axis needs less travel, splitting it between both.
	if overX/minDX < overY/minDY {
		s := sign(dx, a < b) * overX / 2
		cells[a], cells[b] = Point{X: pa.X - s, Y: pa.Y}, Point{X: pb.X + s, Y: pb.Y}
	} else {
		s := sign(dy, a < b) * overY / 2
		cells[a], cells[b] = Point{X: pa.X, Y: pa.Y - s}, Point{X: pb.X, Y: pb.Y + s}
	}
	return true
}

func sign(v float64, tie bool) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	case tie:
		return 1
	default:
		return -1
	}
}

func clamp(cells map[string]Point, half map[string]float64, w, h int) {
	for id, p := range cells {
		lo := math.Max(half[id]-1, 3)
		p.X = math.Min(math.Max(p.X, lo), math.Max(float64(w)-lo, lo))
		p.Y = math.Min(math.Max(p.Y, 1), math.Max(float64(h)-4, 1))
		cells[id] = p
	}
}
