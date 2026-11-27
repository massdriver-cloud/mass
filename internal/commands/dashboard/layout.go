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
}

// Layout places every environment in a w×h cell box, returning each one's
// node center in dots. The same graph and options always give the same layout.
func Layout(g *Graph, w, h int, opts LayoutOptions) map[string]Point {
	if len(g.Envs) == 0 {
		return map[string]Point{}
	}
	pos := settle(g, opts)
	half := labelHalfWidths(g)
	var cells map[string]Point
	if opts.ProducersOnTop {
		cells = stackLayers(g, pos, half, w, h)
	} else {
		cells = fit(pos, w, h)
	}
	separate(g, cells, half, w, h)
	out := make(map[string]Point, len(cells))
	for id, p := range cells {
		out[id] = Point{X: p.X*2 + 1, Y: p.Y*4 + 2}
	}
	return out
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

const layoutIterations = 400

// settle runs a Fruchterman–Reingold simulation in the unit square.
func settle(g *Graph, opts LayoutOptions) map[string]Point {
	envs := envOrder(g)
	pos := make(map[string]Point, len(envs))
	for i, env := range envs {
		angle := 2 * math.Pi * float64(i) / float64(len(envs))
		r := 0.35 + 0.05*jitter(env.ID)
		pos[env.ID] = Point{X: r * math.Cos(angle), Y: r * math.Sin(angle)}
	}
	k := math.Sqrt(1.0 / float64(len(envs)))
	temp := 0.1
	for range layoutIterations {
		disp := map[string]Point{}
		repel(envs, pos, disp, k)
		attract(g, pos, disp, k)
		if opts.GroupByProject {
			cluster(envs, pos, disp, k)
		}
		for _, env := range envs {
			p, d := pos[env.ID], disp[env.ID]
			dist := math.Hypot(p.X, p.Y)
			d.X -= p.X * dist * 4 / k
			d.Y -= p.Y * dist * 4 / k
			length := math.Max(math.Hypot(d.X, d.Y), 1e-9)
			step := math.Min(length, temp)
			pos[env.ID] = Point{X: p.X + d.X/length*step, Y: p.Y + d.Y/length*step}
		}
		temp = math.Max(temp*0.985, 0.002)
	}
	return pos
}

func push(disp map[string]Point, id string, dx, dy float64) {
	d := disp[id]
	disp[id] = Point{X: d.X + dx, Y: d.Y + dy}
}

func delta(a, b Point) (float64, float64, float64) {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx, dy, math.Max(math.Hypot(dx, dy), 1e-3)
}

func repel(envs []*EnvNode, pos, disp map[string]Point, k float64) {
	for i, a := range envs {
		for _, b := range envs[i+1:] {
			dx, dy, d := delta(pos[a.ID], pos[b.ID])
			f := k * k / d
			push(disp, a.ID, dx/d*f, dy/d*f)
			push(disp, b.ID, -dx/d*f, -dy/d*f)
		}
	}
}

func attract(g *Graph, pos, disp map[string]Point, k float64) {
	for _, l := range g.EnvLinks {
		dx, dy, d := delta(pos[l.From], pos[l.To])
		f := d * d / k
		push(disp, l.From, -dx/d*f, -dy/d*f)
		push(disp, l.To, dx/d*f, dy/d*f)
	}
}

func cluster(envs []*EnvNode, pos, disp map[string]Point, k float64) {
	for i, a := range envs {
		for _, b := range envs[i+1:] {
			if a.ProjectID == "" || a.ProjectID != b.ProjectID {
				continue
			}
			dx, dy, d := delta(pos[a.ID], pos[b.ID])
			f := 1.5 * d * d / k
			push(disp, a.ID, -dx/d*f, -dy/d*f)
			push(disp, b.ID, dx/d*f, dy/d*f)
		}
	}
}

// stackLayers puts each layer of producers on its own band of rows, keeping
// the left-to-right order the simulation found and wrapping a layer onto more
// rows when its labels will not fit across.
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

// labelRows is how many cell rows a node and its two-line label take.
const labelRows = 5

// fit scales positions into the cell box, leaving room for labels at the edges.
func fit(pos map[string]Point, w, h int) map[string]Point {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range pos {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
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

// separate nudges nodes whose labels would overlap apart, in cells.
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

// clamp keeps every node and its label on the canvas.
func clamp(cells map[string]Point, half map[string]float64, w, h int) {
	for id, p := range cells {
		lo := math.Max(half[id]-1, 3)
		p.X = math.Min(math.Max(p.X, lo), math.Max(float64(w)-lo, lo))
		p.Y = math.Min(math.Max(p.Y, 1), math.Max(float64(h)-4, 1))
		cells[id] = p
	}
}
