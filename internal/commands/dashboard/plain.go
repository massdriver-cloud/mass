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
		return cmp.Compare(byProject[a][0].ProjectName+a, byProject[b][0].ProjectName+b)
	})
	out := make([][]*EnvNode, 0, len(projects))
	for _, p := range projects {
		envs := byProject[p]
		slices.SortFunc(envs, func(a, b *EnvNode) int { return cmp.Compare(a.Name+a.ID, b.Name+b.ID) })
		out = append(out, envs)
	}
	return out
}

func plainCells(g *Graph, w, h int) map[string][2]int {
	cols := max(1, (w+plainGap)/(plainColumn+plainGap))
	colW := (w - (cols-1)*plainGap) / cols
	cells := map[string][2]int{}
	col, row := 0, 1
	for _, envs := range plainRows(g) {
		need := len(envs) + 2
		if row > 1 && row+need > h && col < cols-1 {
			col, row = col+1, 1
		}
		row++
		for _, env := range envs {
			cells[env.ID] = [2]int{col*(colW+plainGap) + 2, row}
			row++
		}
		row++
	}
	return cells
}

// PlainLayout places environments for the plain theme, in dots, so moving
// the selection with the arrow keys follows the lines.
func PlainLayout(g *Graph, w, h int) map[string]Point {
	out := map[string]Point{}
	for id, c := range plainCells(g, w, h) {
		out[id] = Point{X: float64(c[0]*2 + 1), Y: float64(c[1]*4 + 2)}
	}
	return out
}

func (s Scene) renderPlain(c *Canvas) {
	cols := max(1, (c.W+plainGap)/(plainColumn+plainGap))
	colW := (c.W - (cols-1)*plainGap) / cols
	cells := plainCells(s.Graph, c.W, c.H)
	for _, envs := range plainRows(s.Graph) {
		first := cells[envs[0].ID]
		header := cmp.Or(envs[0].ProjectName, envs[0].ProjectID, "Beyond your access")
		color := projectColor(s.Graph, envs[0].ProjectID)
		c.Text(first[0]-2, first[1]-1, clip(header, colW), lipgloss.NewStyle().Bold(true).Foreground(color))
		if rule := min(colW, plainColumn) - runewidth.StringWidth(header) - 1; rule > 0 {
			c.Text(first[0]-2+runewidth.StringWidth(header)+1, first[1]-1, strings.Repeat("─", rule), styleFaint)
		}
		for _, env := range envs {
			at := cells[env.ID]
			s.plainLine(c, env, at[0], at[1], colW-2)
		}
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
		dot, dotColor = spinner[(s.Frame/3)%len(spinner)], colorBlue
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
		c.Text(col, y, part.text, part.style)
		col += runewidth.StringWidth(part.text) + 2
	}
}

type plainFact struct {
	text  string
	style lipgloss.Style
}

func (s Scene) plainFacts(env *EnvNode) []plainFact {
	switch {
	case env.Unloaded:
		return nil
	case env.Restricted:
		return []plainFact{{"no access", styleFaint}}
	}
	var out []plainFact
	if n := env.FiringCount(); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("▲ %d alarming", n), styleBoldRed})
	}
	if n := len(env.Deploying()); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("⟳ %d deploying", n), lipgloss.NewStyle().Foreground(colorBlue)})
	}
	if n := env.StatusCount(StatusFailed); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("✕ %d failed", n), styleRed})
	}
	ok := env.StatusCount(StatusProvisioned)
	out = append(out, plainFact{fmt.Sprintf("%d/%d up", ok, len(env.Instances)), styleMuted})
	if n := env.Upgrades(); n > 0 {
		out = append(out, plainFact{fmt.Sprintf("↑ %d", n), styleUpgrade})
	}
	if env.Banner != nil {
		out = append(out, plainFact{"⚑", lipgloss.NewStyle().Foreground(levelColor(env.Banner.Level))})
	}
	if s.ShowCost && env.Currency != "" {
		out = append(out, plainFact{Money(env.Cost, env.Currency) + "/mo", styleMuted})
	}
	return out
}
