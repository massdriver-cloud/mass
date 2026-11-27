package dashboard

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tab is one page of the dashboard.
type Tab int

const (
	// TabConstellation shows environments linked by the resources they share.
	TabConstellation Tab = iota
	// TabRuntimes shows how far applications are behind their runtimes.
	TabRuntimes
	// TabAlarms lists every firing alarm.
	TabAlarms
	// TabUpgrades lists every instance with a newer bundle version available.
	TabUpgrades
)

var tabNames = []string{"Constellation", "Runtimes", "Alarms", "Upgrades"}

const (
	frameInterval = 80 * time.Millisecond
	introFrames   = 22
	eventBacklog  = time.Hour
	eventsKept    = 40
	toastFrames   = 50
)

// Links builds web app URLs for what the dashboard shows.
type Links interface {
	Environment(id string) string
	Instance(id string) string
	Runtime(id string) string
}

// Config configures a [Model].
type Config struct {
	// Context cancels reads still in flight when the dashboard closes.
	Context context.Context //nolint:containedctx // the model outlives any one call
	Org     string
	// Refresh is how often to re-read the organization; zero reads once.
	Refresh time.Duration
	// EventPoll is how often to read new audit log events; zero never does.
	EventPoll time.Duration
	Now       func() time.Time
	Links     Links
	// Open opens a URL in the browser.
	Open func(url string) error
}

type (
	snapshotMsg struct {
		snap *Snapshot
		err  error
		at   time.Time
	}
	eventsMsg struct {
		log     Log
		err     error
		initial bool
	}
	openedMsg struct {
		url string
		err error
	}
	frameMsg     time.Time
	refreshMsg   struct{}
	eventPollMsg struct{}
)

// Model is the dashboard's bubbletea model.
type Model struct {
	api      API
	cfg      Config
	snap     *Snapshot
	graph    *Graph
	err      error
	loading  bool
	fetched  time.Time
	width    int
	height   int
	tab      Tab
	frame    int
	selected string
	blastOn  bool
	showApps bool
	grouped  bool
	stacked  bool
	showCost bool
	help     bool

	searching bool
	filter    string

	runtimeIdx int
	bump       Bump
	alarmIdx   int
	upgradeIdx int

	replay        replay
	replayRequest int
	introAt       int
	events        []Event
	seen          map[string]time.Time
	since         time.Time
	flashes       map[string]Flash
	toast         string
	toastAt       int
	pos           map[string]Point
	layoutKey     string
}

// NewModel returns a dashboard that reads from api.
func NewModel(api API, cfg Config) Model {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return Model{
		api: api, cfg: cfg, loading: true, blastOn: true, showApps: true, introAt: -1,
		seen: map[string]time.Time{}, flashes: map[string]Flash{}, since: cfg.Now().Add(-eventBacklog),
	}
}

// Init starts the first reads and the animation clock.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.fetch(), frame()}
	if m.cfg.EventPoll > 0 {
		cmds = append(cmds, m.fetchEvents(true))
	}
	return tea.Batch(cmds...)
}

func frame() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return frameMsg(t) })
}

func (m Model) fetch() tea.Cmd {
	api, ctx, now := m.api, m.ctx(), m.cfg.Now
	return func() tea.Msg {
		snap, err := Fetch(ctx, api)
		return snapshotMsg{snap: snap, err: err, at: now()}
	}
}

func (m Model) fetchEvents(initial bool) tea.Cmd {
	api, ctx, since := m.api, m.ctx(), m.since
	return func() tea.Msg {
		log, err := api.Events(ctx, since)
		return eventsMsg{log: log, err: err, initial: initial}
	}
}

func (m Model) ctx() context.Context {
	if m.cfg.Context == nil {
		return context.Background()
	}
	return m.cfg.Context
}

func after(d time.Duration, msg tea.Msg) tea.Cmd {
	if d <= 0 {
		return nil
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return msg })
}

// Update handles one message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	if next.graph != nil {
		next = next.withLayout(next.canvasSize())
	}
	return next, cmd
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case frameMsg:
		return m.tick(), frame()
	case refreshMsg:
		return m.reload()
	case eventPollMsg:
		return m, m.fetchEvents(false)
	case snapshotMsg:
		return m.applySnapshot(msg)
	case eventsMsg:
		return m.applyEvents(msg)
	case historyMsg:
		return m.applyHistory(msg), nil
	case openedMsg:
		m.toast, m.toastAt = "↗ opened "+msg.url, m.frame
		if msg.err != nil {
			m.toast = "↗ could not open a browser: " + msg.url
		}
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) tick() Model {
	m.frame++
	flashes := make(map[string]Flash, len(m.flashes))
	for id, f := range m.flashes {
		if f.Age+1 < FlashFrames {
			flashes[id] = Flash{Age: f.Age + 1, Label: f.Label}
		}
	}
	m.flashes = flashes
	return m.advanceReplay()
}

func (m Model) reload() (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	return m, m.fetch()
}

func (m Model) applySnapshot(msg snapshotMsg) (Model, tea.Cmd) {
	m.loading = false
	m.err = msg.err
	if msg.err == nil {
		m.snap, m.graph, m.fetched = msg.snap, BuildGraph(msg.snap), msg.at
		m.layoutKey = ""
		if m.introAt < 0 {
			m.introAt = m.frame
		}
		if m.graph.Env(m.selected) == nil {
			m.selected = m.firstSelection()
		}
		m.runtimeIdx = min(m.runtimeIdx, max(len(m.snap.Runtimes)-1, 0))
		m.alarmIdx = min(m.alarmIdx, max(len(m.graph.Firing)-1, 0))
		m.upgradeIdx = min(m.upgradeIdx, max(len(m.upgrades())-1, 0))
	}
	return m, after(m.cfg.Refresh, refreshMsg{})
}

// applyEvents re-reads the organization on a deployment event so rings and
// spinners catch up with it.
func (m Model) applyEvents(msg eventsMsg) (Model, tea.Cmd) {
	poll := after(m.cfg.EventPoll, eventPollMsg{})
	if msg.err != nil {
		return m, poll
	}
	if msg.log.Truncated && !msg.initial {
		m.toast, m.toastAt = fmt.Sprintf("more than %d events since the last check; showing the newest", len(msg.log.Events)), m.frame
	}
	fresh := make([]Event, 0, len(msg.log.Events))
	for _, ev := range msg.log.Events {
		if _, ok := m.seen[ev.ID]; !ok {
			m.seen[ev.ID] = ev.At
			fresh = append(fresh, ev)
		}
		if ev.At.After(m.since) {
			m.since = ev.At
		}
	}
	// Reads start at since inclusive, so only events at since can come back.
	for id, at := range m.seen {
		if at.Before(m.since) {
			delete(m.seen, id)
		}
	}
	slices.SortStableFunc(fresh, func(a, b Event) int { return b.At.Compare(a.At) })
	m.events = append(fresh, m.events...)
	m.events = m.events[:min(len(m.events), eventsKept)]
	if msg.initial || m.replay.on {
		return m, poll
	}
	flashes := make(map[string]Flash, len(m.flashes)+len(fresh))
	for id, f := range m.flashes {
		flashes[id] = f
	}
	deployed := false
	for _, ev := range fresh {
		if ev.EnvID != "" {
			flashes[ev.EnvID] = Flash{Label: EventLabel(ev)}
		}
		deployed = deployed || strings.HasPrefix(ev.Type, "deployment.")
	}
	m.flashes = flashes
	if deployed && !m.loading {
		m.loading = true
		return m, tea.Batch(poll, m.fetch())
	}
	return m, poll
}

func (m Model) firstSelection() string {
	if len(m.graph.Firing) > 0 {
		return m.graph.Firing[0].EnvID
	}
	if len(m.graph.Envs) > 0 {
		return m.graph.Envs[0].ID
	}
	return ""
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg), nil
	}
	if m.replay.picking {
		return m.handlePickKey(msg.String())
	}
	switch key := msg.String(); key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "1", "2", "3", "4":
		m.tab = Tab(key[0] - '1')
	case "right", "l", "left", "h", "up", "k", "down", "j", "tab", "shift+tab", "enter":
		m = m.move(key)
	case "r":
		return m.reload()
	case "o":
		return m, m.openSelection()
	case "t":
		if m.tab == TabConstellation {
			m = m.toggleReplay()
		}
	case " ", "[", "]":
		if m.replay.on {
			m = m.handleReplayKey(key)
		}
	case "?":
		m.help = !m.help
	case "esc":
		m.filter, m.help = "", false
	default:
		m = m.handleTabKey(key)
	}
	return m, nil
}

func (m Model) handleTabKey(key string) Model {
	switch m.tab {
	case TabConstellation:
		return m.handleConstellationKey(key)
	case TabRuntimes:
		bumps := map[string]Bump{"0": BumpNone, "p": BumpPatch, "m": BumpMinor, "M": BumpMajor}
		if b, ok := bumps[key]; ok {
			m.bump = b
		}
	case TabAlarms, TabUpgrades:
	}
	return m
}

func (m Model) handleConstellationKey(key string) Model {
	switch key {
	case "b":
		m.blastOn = !m.blastOn
	case "a":
		m.showApps = !m.showApps
	case "c":
		m.showCost = !m.showCost
	case "g":
		m.grouped = !m.grouped
	case "s":
		m.stacked = !m.stacked
	case "/":
		m.searching, m.filter = true, ""
	}
	return m
}

func (m Model) handleSearchKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter":
		m.searching = false
		return m.selectFirstMatch()
	case "esc":
		m.searching, m.filter = false, ""
	case "backspace":
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.filter += string(msg.Runes)
			return m.selectFirstMatch()
		}
	}
	return m
}

func (m Model) selectFirstMatch() Model {
	if m.graph == nil {
		return m
	}
	s := Scene{Filter: m.filter}
	for _, env := range m.orderedEnvs() {
		if s.Matches(env) {
			m.selected = env.ID
			break
		}
	}
	return m
}

func (m Model) move(key string) Model {
	if m.graph == nil {
		return m
	}
	switch m.tab {
	case TabConstellation:
		return m.moveSelection(key)
	case TabRuntimes:
		m.runtimeIdx = step(m.runtimeIdx, len(m.snap.Runtimes), key)
	case TabAlarms:
		if key == "enter" && len(m.graph.Firing) > 0 {
			return m.focus(m.graph.Firing[m.alarmIdx].EnvID)
		}
		m.alarmIdx = step(m.alarmIdx, len(m.graph.Firing), key)
	case TabUpgrades:
		ups := m.upgrades()
		if key == "enter" && len(ups) > 0 {
			return m.focus(ups[m.upgradeIdx].EnvID)
		}
		m.upgradeIdx = step(m.upgradeIdx, len(ups), key)
	}
	return m
}

func (m Model) focus(envID string) Model {
	m.tab, m.selected, m.blastOn = TabConstellation, envID, true
	return m
}

func step(idx, n int, key string) int {
	if n == 0 {
		return 0
	}
	switch key {
	case "down", "j", "tab", "right", "l":
		return (idx + 1) % n
	case "up", "k", "shift+tab", "left", "h":
		return (idx - 1 + n) % n
	}
	return idx
}

func (m Model) orderedEnvs() []*EnvNode {
	envs := slices.Clone(m.graph.Envs)
	slices.SortStableFunc(envs, func(a, b *EnvNode) int {
		pa, pb := m.pos[a.ID], m.pos[b.ID]
		if math.Abs(pa.Y-pb.Y) > 4 {
			return int(math.Copysign(1, pa.Y-pb.Y))
		}
		return int(math.Copysign(1, pa.X-pb.X))
	})
	return envs
}

var directions = map[string]Point{
	"right": {X: 1}, "l": {X: 1}, "left": {X: -1}, "h": {X: -1},
	"up": {Y: -1}, "k": {Y: -1}, "down": {Y: 1}, "j": {Y: 1},
}

func (m Model) moveSelection(key string) Model {
	envs := m.orderedEnvs()
	if len(envs) == 0 {
		return m
	}
	if dir, ok := directions[key]; ok {
		if id := m.nearest(dir); id != "" {
			m.selected = id
		}
		return m
	}
	idx := slices.IndexFunc(envs, func(e *EnvNode) bool { return e.ID == m.selected })
	m.selected = envs[step(idx, len(envs), key)].ID
	return m
}

// nearest finds the closest environment in a direction, favoring ones that
// sit squarely along it over ones off to the side.
func (m Model) nearest(dir Point) string {
	from, ok := m.pos[m.selected]
	if !ok {
		return ""
	}
	best, bestScore := "", math.Inf(1)
	for id, p := range m.pos {
		dx, dy := p.X-from.X, p.Y-from.Y
		along := dx*dir.X + dy*dir.Y
		if id == m.selected || along <= 0 {
			continue
		}
		across := math.Abs(dx*dir.Y - dy*dir.X)
		if score := along + 2*across; score < bestScore || (score == bestScore && id < best) {
			best, bestScore = id, score
		}
	}
	return best
}

func (m Model) selectionURL() string {
	if m.cfg.Links == nil || m.graph == nil {
		return ""
	}
	switch m.tab {
	case TabRuntimes:
		if len(m.snap.Runtimes) > 0 {
			return m.cfg.Links.Runtime(m.snap.Runtimes[m.runtimeIdx].ID)
		}
	case TabAlarms:
		if len(m.graph.Firing) > 0 {
			return m.cfg.Links.Instance(m.graph.Firing[m.alarmIdx].InstanceID)
		}
	case TabUpgrades:
		if ups := m.upgrades(); len(ups) > 0 {
			return m.cfg.Links.Instance(ups[m.upgradeIdx].ID)
		}
	case TabConstellation:
		if env := m.graph.Env(m.selected); env != nil && !env.Restricted {
			return m.cfg.Links.Environment(env.ID)
		}
	}
	return ""
}

func (m Model) openSelection() tea.Cmd {
	url, open := m.selectionURL(), m.cfg.Open
	if url == "" || open == nil {
		return nil
	}
	return func() tea.Msg { return openedMsg{url: url, err: open(url)} }
}

func (m Model) blast() *Blast {
	if !m.blastOn || m.graph == nil || m.selected == "" {
		return nil
	}
	env := m.graph.Env(m.selected)
	if env == nil {
		return nil
	}
	var sources []string
	for _, a := range env.Firing {
		sources = append(sources, a.InstanceID)
	}
	if len(sources) == 0 {
		sources = []string{env.ID}
	}
	return m.graph.BlastRadius(sources...)
}

func (m Model) upgrades() []*InstNode {
	if m.graph == nil {
		return nil
	}
	var out []*InstNode
	for _, env := range m.graph.Envs {
		for _, inst := range env.Instances {
			if inst.HasUpgrade() {
				out = append(out, inst)
			}
		}
	}
	slices.SortFunc(out, func(a, b *InstNode) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func (m Model) intro() float64 {
	if m.introAt < 0 {
		return 0
	}
	return math.Min(float64(m.frame-m.introAt)/introFrames, 1)
}

func (m Model) panelWidth() int {
	switch {
	case m.width >= 120:
		return 46
	case m.width >= 90:
		return 36
	default:
		return 0
	}
}

func (m Model) bodyHeight() int { return max(m.height-4, 3) }

func (m Model) canvasSize() (int, int) {
	w := m.width
	if pw := m.panelWidth(); pw > 0 {
		w -= pw + 3
	}
	return max(w, 10), m.bodyHeight()
}

func (m Model) withLayout(w, h int) Model {
	key := fmt.Sprintf("%d:%d:%t:%t", w, h, m.grouped, m.stacked)
	if key == m.layoutKey && m.pos != nil {
		return m
	}
	m.pos = Layout(m.graph, w, h, LayoutOptions{GroupByProject: m.grouped, ProducersOnTop: m.stacked})
	m.layoutKey = key
	return m
}

// Selected is the id of the selected environment.
func (m Model) Selected() string { return m.selected }

// ActiveTab is the tab on screen.
func (m Model) ActiveTab() Tab { return m.tab }
