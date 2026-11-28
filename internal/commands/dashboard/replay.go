package dashboard

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReplayWindows are how far back a replay can start.
var ReplayWindows = []time.Duration{
	5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 30 * time.Minute, 45 * time.Minute,
	time.Hour, 2 * time.Hour, 3 * time.Hour, 4 * time.Hour, 6 * time.Hour, 8 * time.Hour,
	12 * time.Hour, 16 * time.Hour, 24 * time.Hour,
}

const (
	defaultReplayWindow = time.Hour
	replayStepFrames    = 2
	replayMaxSpeed      = 16
	// replaySteps is about how many steps a replay takes at its starting
	// speed, so any window plays back in under a minute.
	replaySteps = 300
)

type replay struct {
	picking bool
	window  time.Duration
	on      bool
	loading bool
	paused  bool
	err     error
	log     Log
	pos     int
	speed   int
}

// historyMsg carries the request it answers, so a replay cancelled while
// loading cannot take the answer meant for the one that replaced it.
type historyMsg struct {
	log     Log
	err     error
	request int
}

func (m Model) toggleReplay() Model {
	window := m.replay.window
	if window == 0 {
		window = defaultReplayWindow
	}
	if !m.replay.on && !m.replay.loading && m.replay.err == nil && !m.replay.picking {
		m.replay = replay{picking: true, window: window}
		return m
	}
	m.replay = replay{window: window}
	m.flashes = map[string]Flash{}
	return m
}

func (m Model) handlePickKey(key string) (Model, tea.Cmd) {
	idx := slices.Index(ReplayWindows, m.replay.window)
	switch key {
	case "left", "h":
		m.replay.window = ReplayWindows[max(idx-1, 0)]
	case "right", "l":
		m.replay.window = ReplayWindows[min(idx+1, len(ReplayWindows)-1)]
	case "enter", " ":
		return m.startReplay()
	case "esc", "t":
		m.replay = replay{window: m.replay.window}
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) startReplay() (Model, tea.Cmd) {
	m.replay = replay{window: m.replay.window, loading: true, speed: 1}
	m.replayRequest++
	return m, m.readHistory()
}

func (m Model) readHistory() tea.Cmd {
	api, ctx, since, request := m.api, m.ctx(), m.cfg.Now().Add(-m.replay.window), m.replayRequest
	return func() tea.Msg {
		log, err := api.History(ctx, since)
		return historyMsg{log: log, err: err, request: request}
	}
}

func (m Model) applyHistory(msg historyMsg) Model {
	if !m.replay.loading || msg.request != m.replayRequest {
		return m
	}
	m.replay.loading, m.replay.err = false, msg.err
	if msg.err == nil {
		m.replay.on, m.replay.log, m.flashes = true, msg.log, map[string]Flash{}
		m.replay.speed = startSpeed(len(msg.log.Events))
	}
	return m
}

func startSpeed(n int) int {
	speed := 1
	for speed < replayMaxSpeed && speed*replaySteps < n {
		speed *= 2
	}
	return speed
}

// WindowLabel names a replay window, like 45m or 6h.
func WindowLabel(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func windowWords(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d == time.Hour {
		return "hour"
	}
	return fmt.Sprintf("%d hours", int(d.Hours()))
}

func (m Model) handleReplayKey(key string) Model {
	switch key {
	case " ":
		if m.replay.pos >= len(m.replay.log.Events) {
			m.replay.pos = 0
		}
		m.replay.paused = !m.replay.paused
	case "]":
		m.replay.speed = min(m.replay.speed*2, replayMaxSpeed)
	case "[":
		m.replay.speed = max(m.replay.speed/2, 1)
	}
	return m
}

func (m Model) advanceReplay() Model {
	r := m.replay
	if !r.on || r.paused || m.frame%replayStepFrames != 0 {
		return m
	}
	for range r.speed {
		if r.pos >= len(r.log.Events) {
			r.paused = true
			break
		}
		if ev := r.log.Events[r.pos]; ev.EnvID != "" {
			m.flashes[ev.EnvID] = Flash{Label: EventLabel(ev), Emoji: EventEmoji(ev.Type)}
		}
		r.pos++
	}
	m.replay = r
	return m
}

func (m Model) replayTicker() string {
	r := m.replay
	badge := lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(colorOrange).Render(" ⏪ REPLAY ")
	switch {
	case r.picking:
		return badge + " " + windowPicker(r.window)
	case r.loading:
		return badge + styleMuted.Render(" reading the last "+windowWords(r.window)+"…")
	case r.err != nil:
		return badge + styleRed.Render(" could not read the audit log: "+r.err.Error())
	case len(r.log.Events) == 0:
		return badge + styleMuted.Render(" nothing happened in the last "+windowWords(r.window))
	}
	clock := r.log.Events[max(r.pos-1, 0)].At.Local().Format("Mon 15:04")
	state := fmt.Sprintf("×%d", r.speed)
	if r.paused {
		state = "paused"
	}
	if r.pos >= len(r.log.Events) {
		state = "done"
	}
	parts := []string{
		badge,
		styleTitle.Render(clock),
		progress(r.pos, len(r.log.Events), 24),
		styleMuted.Render(fmt.Sprintf("%d/%d %s", r.pos, len(r.log.Events), state)),
	}
	if r.log.Truncated {
		parts = append(parts, styleUpgrade.Render(fmt.Sprintf("newest %d only", len(r.log.Events))))
	}
	for i := r.pos - 1; i >= 0 && i >= r.pos-4; i-- {
		parts = append(parts, m.tickerItem(r.log.Events[i], r.log.Events[max(r.pos-1, 0)].At))
	}
	return strings.Join(parts, "  ")
}

func windowPicker(selected time.Duration) string {
	parts := []string{styleMuted.Render("how far back?")}
	for _, w := range ReplayWindows {
		label := " " + WindowLabel(w) + " "
		if w == selected {
			parts = append(parts, styleSelected.Render(label))
		} else {
			parts = append(parts, styleMuted.Render(label))
		}
	}
	return strings.Join(parts, "")
}

func progress(done, total, w int) string {
	filled := done * w / max(total, 1)
	return styleTeal.Render(strings.Repeat("━", filled)) + styleFaint.Render(strings.Repeat("─", w-filled))
}
