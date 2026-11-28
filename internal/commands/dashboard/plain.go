package dashboard

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ThemePlain lists environments by project, one line each, with nothing
// moving: the quickest read of what is healthy and what needs a look.
const ThemePlain = "plain"

const (
	plainColumn = 56
	plainGap    = 4
)

func plainRows(g *Graph) [][]*EnvNode {
	byProject := map[string][]*EnvNode{}
	var projects []string
	for _, env := range g.Envs {
		if _, ok := byProject[env.ProjectID]; !ok {
			projects = append(projects, env.ProjectID)
		}
		byProject[env.ProjectID] = append(byProject[env.ProjectID], env)
	}
	slices.SortFunc(projects, func(a, b string) int {
		if (a == "") != (b == "") {
			return cmp.Compare(len(b), len(a))
		}
		return cmp.Or(cmp.Compare(byProject[a][0].ProjectName, byProject[b][0].ProjectName), cmp.Compare(a, b))
	})
	out := make([][]*EnvNode, 0, len(projects))
	for _, p := range projects {
		envs := byProject[p]
		slices.SortFunc(envs, func(a, b *EnvNode) int { return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID)) })
		out = append(out, envs)
	}
	return out
}

// plainSheet is where the plain theme puts things on a w×h canvas.
type plainSheet struct {
	colW    int
	headers map[string][2]int
	cells   map[string][2]int
	// more counts the environments that did not fit, listed at moreAt.
	more   int
	moreAt [2]int
}

func newPlainSheet(g *Graph, w, h int) plainSheet {
	cols := max(1, (w+plainGap)/(plainColumn+plainGap))
	sheet := plainSheet{colW: (w - (cols-1)*plainGap) / cols, headers: map[string][2]int{}, cells: map[string][2]int{}}
	col, row := 0, 1
	last := h - 1
	for _, envs := range plainRows(g) {
		if row > 1 && row+len(envs)+2 > h && col < cols-1 {
			col, row = col+1, 1
		}
		x := col * (sheet.colW + plainGap)
		for i, env := range envs {
			if row+1 >= last {
				sheet.more++
				continue
			}
			if i == 0 {
				sheet.headers[env.ProjectID] = [2]int{x, row}
				row++
			}
			sheet.cells[env.ID] = [2]int{x + 2, row}
			row++
		}
		if sheet.more == 0 {
			row++
		}
		sheet.moreAt = [2]int{x + 2, min(row, last)}
	}
	return sheet
}

// PlainLayout places environments for the plain theme, in dots, so moving
// the selection with the arrow keys follows the lines. Environments that do
// not fit are left out.
func PlainLayout(g *Graph, w, h int) map[string]Point {
	out := map[string]Point{}
	for id, c := range newPlainSheet(g, w, h).cells {
		out[id] = Point{X: float64(c[0]*2 + 1), Y: float64(c[1]*4 + 2)}
	}
	return out
}

func (s Scene) renderPlain(c *Canvas) {
	sheet := newPlainSheet(s.Graph, c.W, c.H)
	for _, envs := range plainRows(s.Graph) {
		if at, ok := sheet.headers[envs[0].ProjectID]; ok {
			header := cmp.Or(envs[0].ProjectName, envs[0].ProjectID, "Beyond your access")
			color := projectColor(s.Graph, envs[0].ProjectID)
			c.Text(at[0], at[1], clip(header, sheet.colW), lipgloss.NewStyle().Bold(true).Foreground(color))
			if rule := min(sheet.colW, plainColumn) - runewidth.StringWidth(header) - 1; rule > 0 {
				c.Text(at[0]+runewidth.StringWidth(header)+1, at[1], strings.Repeat("─", rule), styleFaint)
			}
		}
		for _, env := range envs {
			if at, ok := sheet.cells[env.ID]; ok {
				s.plainLine(c, env, at[0], at[1], sheet.colW-2)
			}
		}
	}
	if sheet.more > 0 {
		c.Text(sheet.moreAt[0], sheet.moreAt[1], fmt.Sprintf("+%d more · / finds any environment", sheet.more), styleFaint)
	}
}

func (x *svg) plain(w, h int) {
	s := x.scene
	sheet := newPlainSheet(s.Graph, w, h)
	cell := func(c [2]int) Point { return Point{X: float64(c[0] * 2), Y: float64(c[1]*4 + 3)} }
	fs := x.font(3.2)
	advance := func(text string) float64 { return float64(runewidth.StringWidth(text)) * fs * 0.62 / x.k }
	for _, envs := range plainRows(s.Graph) {
		if at, ok := sheet.headers[envs[0].ProjectID]; ok {
			header := cmp.Or(envs[0].ProjectName, envs[0].ProjectID, "Beyond your access")
			x.text(cell(at), 0, 0, header, fs, projectColor(s.Graph, envs[0].ProjectID), "start", true)
		}
		for _, env := range envs {
			at, ok := sheet.cells[env.ID]
			if !ok {
				continue
			}
			p := cell(at)
			color := statusColor(env.Health())
			if env.Unloaded || env.Restricted {
				color = colorFaint
			}
			x.text(p, 0, 0, "●", fs, color, "start", false)
			name, nameColor := env.Name, colorText
			if env.ID == s.Selected {
				nameColor = colorTeal
			}
			x.text(p, 6*x.k, 0, name, fs, nameColor, "start", env.ID == s.Selected)
			col := 6 + float64(min(24, max((sheet.colW-2)/2, 10)))*2
			for _, f := range s.plainFacts(env) {
				x.text(p, col*x.k, 0, f.text, fs, f.color, "start", f.bold)
				col += advance(f.text) + 4
			}
		}
	}
	if sheet.more > 0 {
		x.text(cell(sheet.moreAt), 0, 0, fmt.Sprintf("+%d more", sheet.more), fs, colorFaint, "start", false)
	}
}

func (s Scene) plainLine(c *Canvas, env *EnvNode, x, y, w int) {
	dot, dotColor := "●", statusColor(env.Health())
	switch {
	case env.Unloaded:
		dot, dotColor = "🔭", colorFaint
	case env.Restricted:
		dot, dotColor = "◌", colorFaint
	case len(env.Deploying()) > 0:
		dot, dotColor = "⟳", colorBlue
	}
	c.Text(x, y, dot, lipgloss.NewStyle().Foreground(dotColor))
	name, _ := s.labelStyles(env)
	if hop, ok := s.Blast.Reaches(env.ID); ok && env.ID != s.Selected && hop > 0 {
		name = lipgloss.NewStyle().Bold(true).Foreground(hopColor(hop))
	}
	label := env.Name
	if env.ID == s.Selected {
		label = " " + label + " "
	}
	nameW := min(24, max(w/2, 10))
	c.Text(x+3, y, clip(label, nameW), name)
	col := x + 3 + nameW + 1
	for _, part := range s.plainFacts(env) {
		if col+runewidth.StringWidth(part.text) > x+w {
			break
		}
		c.Text(col, y, part.text, lipgloss.NewStyle().Bold(part.bold).Foreground(part.color))
		col += runewidth.StringWidth(part.text) + 2
	}
}

type plainFact struct {
	text  string
	color lipgloss.Color
	bold  bool
}

func (s Scene) plainFacts(env *EnvNode) []plainFact {
	switch {
	case env.Unloaded:
		return nil
	case env.Restricted:
		return []plainFact{{text: "no access", color: colorFaint}}
	}
	var out []plainFact
	if n := env.FiringCount(); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("▲ %d alarming", n), colorRed, true})
	}
	if n := len(env.Deploying()); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("⟳ %d deploying", n), colorBlue, false})
	}
	if n := env.StatusCount(StatusFailed); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("✕ %d failed", n), colorRed, false})
	}
	if len(env.Instances) > 0 {
		ok := env.StatusCount(StatusProvisioned)
		out = append(out, plainFact{fmt.Sprintf("%d/%d up", ok, len(env.Instances)), colorMuted, false})
	}
	if n := env.Upgrades(); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("↑ %d", n), colorYellow, true})
	}
	if env.Banner != nil {
		out = append(out, plainFact{"⚑", levelColor(env.Banner.Level), false})
	}
	if s.ShowCost && env.Currency != "" {
		out = append(out, plainFact{Money(env.Cost, env.Currency) + "/mo", colorMuted, false})
	}
	return out
}
