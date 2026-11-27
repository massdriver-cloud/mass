package dashboard

import (
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Canvas is a grid of terminal cells drawn with braille dots, two dots across
// and four down per cell, so lines and rings come out four times finer than
// the cells. Text written to a cell replaces its dots.
type Canvas struct {
	W, H   int
	cells  []cell
	styles []lipgloss.Style
}

type cell struct {
	dots  uint8
	color lipgloss.Color
	layer int
	text  rune
	style int
	isTxt bool
	cont  bool
}

// braille dot bits, indexed [x][y] within a cell.
var brailleBits = [2][4]uint8{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

// NewCanvas returns an empty w×h canvas.
func NewCanvas(w, h int) *Canvas {
	w, h = max(w, 1), max(h, 1)
	return &Canvas{W: w, H: h, cells: make([]cell, w*h)}
}

// DotW and DotH are the canvas size in dots.
func (c *Canvas) DotW() int { return c.W * 2 }

// DotH is the canvas height in dots.
func (c *Canvas) DotH() int { return c.H * 4 }

func (c *Canvas) at(cx, cy int) *cell {
	if cx < 0 || cy < 0 || cx >= c.W || cy >= c.H {
		return nil
	}
	return &c.cells[cy*c.W+cx]
}

// Dot sets the dot at (x, y). A cell takes the color of the highest layer
// drawn into it, so links stay beneath nodes.
func (c *Canvas) Dot(x, y int, color lipgloss.Color, layer int) {
	if x < 0 || y < 0 {
		return
	}
	ce := c.at(x/2, y/4)
	if ce == nil || ce.isTxt {
		return
	}
	ce.dots |= brailleBits[x%2][y%4]
	if layer >= ce.layer || ce.color == "" {
		ce.color, ce.layer = color, layer
	}
}

// Line draws a straight line of dots, skipping dots per pattern: every dot is
// drawn when dash is 0, otherwise dash dots on and dash dots off.
func (c *Canvas) Line(x0, y0, x1, y1 float64, color lipgloss.Color, layer, dash int) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= steps; i++ {
		if dash > 0 && (i/dash)%2 == 1 {
			continue
		}
		t := float64(i) / float64(steps)
		c.Dot(int(math.Round(x0+(x1-x0)*t)), int(math.Round(y0+(y1-y0)*t)), color, layer)
	}
}

// Stroke styles a curve.
type Stroke struct {
	Color lipgloss.Color
	Layer int
	// Dash draws Dash dots on and Dash dots off; zero draws a solid line.
	Dash int
	// Gap leaves out the stretch within Gap dots of either end.
	Gap float64
	// Until stops the curve that fraction of the way along; zero draws all of it.
	Until float64
}

// Curve draws a quadratic Bézier from p0 to p2 bent toward p1.
func (c *Canvas) Curve(p0, p1, p2 Point, st Stroke) {
	length := p0.Dist(p1) + p1.Dist(p2)
	steps := max(int(length), 2)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		if (st.Dash > 0 && (i/st.Dash)%2 == 1) || (st.Until > 0 && t > st.Until) {
			continue
		}
		p := Bezier(p0, p1, p2, t)
		if p.Dist(p0) < st.Gap || p.Dist(p2) < st.Gap {
			continue
		}
		c.Dot(int(math.Round(p.X)), int(math.Round(p.Y)), st.Color, st.Layer)
	}
}

// Ring draws a circle of radius r dots, coloring each arc by colorAt, which
// receives the angle as a fraction of a full turn starting at twelve o'clock
// and leaves the arc out by returning "".
func (c *Canvas) Ring(center Point, r float64, colorAt func(float64) lipgloss.Color, layer int) {
	steps := max(int(2*math.Pi*r*1.5), 8)
	for i := range steps {
		frac := float64(i) / float64(steps)
		angle := frac*2*math.Pi - math.Pi/2
		x := center.X + r*math.Cos(angle)
		y := center.Y + r*math.Sin(angle)
		if color := colorAt(frac); color != "" {
			c.Dot(int(math.Round(x)), int(math.Round(y)), color, layer)
		}
	}
}

// Disc fills a circle of radius r dots.
func (c *Canvas) Disc(center Point, r float64, color lipgloss.Color, layer int) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r+0.5 {
				c.Dot(int(math.Round(center.X+dx)), int(math.Round(center.Y+dy)), color, layer)
			}
		}
	}
}

// Text writes s starting at cell (cx, cy), clipped to the canvas.
func (c *Canvas) Text(cx, cy int, s string, style lipgloss.Style) {
	c.styles = append(c.styles, style)
	id := len(c.styles)
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if ce := c.at(cx, cy); ce != nil {
			ce.text, ce.style, ce.isTxt, ce.dots = r, id, true, 0
			for i := 1; i < w; i++ {
				if next := c.at(cx+i, cy); next != nil {
					next.isTxt, next.cont = true, true
				}
			}
		}
		cx += w
	}
}

// TextCentered writes s centered on cell column cx.
func (c *Canvas) TextCentered(cx, cy int, s string, style lipgloss.Style) {
	c.Text(cx-runewidth.StringWidth(s)/2, cy, s, style)
}

// Render returns the canvas as styled lines.
func (c *Canvas) Render() string {
	var b strings.Builder
	for y := range c.H {
		if y > 0 {
			b.WriteByte('\n')
		}
		c.renderRow(&b, y)
	}
	return b.String()
}

func (c *Canvas) renderRow(b *strings.Builder, y int) {
	var run strings.Builder
	var runStyle lipgloss.Style
	runKey := ""
	flush := func() {
		if run.Len() > 0 {
			b.WriteString(runStyle.Render(run.String()))
			run.Reset()
		}
	}
	for x := range c.W {
		ce := c.cells[y*c.W+x]
		if ce.cont {
			continue
		}
		ch, style, key := c.glyph(ce)
		if key != runKey {
			flush()
			runStyle, runKey = style, key
		}
		run.WriteRune(ch)
	}
	flush()
}

var plain = lipgloss.NewStyle()

func (c *Canvas) glyph(ce cell) (rune, lipgloss.Style, string) {
	switch {
	case ce.isTxt:
		return ce.text, c.styles[ce.style-1], "t" + strconv.Itoa(ce.style)
	case ce.dots != 0:
		return rune(0x2800) + rune(ce.dots), lipgloss.NewStyle().Foreground(ce.color), "d" + string(ce.color)
	default:
		return ' ', plain, ""
	}
}

// Point is a position in dots.
type Point struct{ X, Y float64 }

// Dist is the distance between p and q.
func (p Point) Dist(q Point) float64 { return math.Hypot(q.X-p.X, q.Y-p.Y) }

// Bezier is the point at t along the quadratic Bézier p0, p1, p2.
func Bezier(p0, p1, p2 Point, t float64) Point {
	u := 1 - t
	return Point{
		X: u*u*p0.X + 2*u*t*p1.X + t*t*p2.X,
		Y: u*u*p0.Y + 2*u*t*p1.Y + t*t*p2.Y,
	}
}

// Bend is the control point that bows the segment a→b to one side by a
// fraction of its length.
func Bend(a, b Point, amount float64) Point {
	mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
	dx, dy := b.X-a.X, b.Y-a.Y
	return Point{X: mx - dy*amount, Y: my + dx*amount}
}
