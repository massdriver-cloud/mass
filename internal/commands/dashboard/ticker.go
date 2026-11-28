package dashboard

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// EventLabel names an event in a few words, like "deployment completed".
func EventLabel(ev Event) string {
	return strings.ReplaceAll(strings.ReplaceAll(ev.Type, ".", " "), "_", " ")
}

func eventColor(ev Event) lipgloss.Color {
	switch {
	case strings.HasSuffix(ev.Type, ".failed"):
		return colorRed
	case strings.HasPrefix(ev.Type, "alarm"):
		return colorOrange
	case strings.HasPrefix(ev.Type, "deployment."):
		return colorBlue
	case strings.HasPrefix(ev.Type, "stash."):
		return colorYellow
	default:
		return colorTeal
	}
}

func (m Model) ticker() string {
	if m.replay.on || m.replay.loading || m.replay.picking || m.replay.err != nil {
		return m.replayTicker()
	}
	if m.cfg.EventPoll <= 0 {
		return ""
	}
	live := styleBoldRed.Render("●")
	if m.frame%16 >= 8 {
		live = styleFaint.Render("●")
	}
	parts := []string{live + styleMuted.Render(" LIVE")}
	if len(m.events) == 0 {
		parts = append(parts, styleFaint.Render("waiting for activity"))
	}
	now := m.cfg.Now()
	for _, ev := range m.events {
		parts = append(parts, m.tickerItem(ev, now))
		if lipgloss.Width(strings.Join(parts, "   ")) > m.width {
			break
		}
	}
	return strings.Join(parts, "   ")
}

func (m Model) tickerItem(ev Event, now time.Time) string {
	label := lipgloss.NewStyle().Foreground(eventColor(ev)).Render(EventLabel(ev))
	if f, ok := m.flashes[ev.EnvID]; ok && f.Age < FlashFrames/2 && f.Label == EventLabel(ev) {
		label = lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(EventLabel(ev))
	}
	where := ev.EnvID
	if m.graph != nil && ev.EnvID != "" {
		where = envLabel(m.graph, ev.EnvID)
	}
	item := styleFaint.Render(Ago(now, ev.At)) + " " + label
	if where != "" {
		item += styleMuted.Render(" " + where)
	}
	if ev.Detail != "" {
		item += styleFaint.Render(" · " + ev.Detail)
	}
	if ev.Actor != "" {
		item += styleFaint.Render(" · " + ev.Actor)
	}
	return item
}
