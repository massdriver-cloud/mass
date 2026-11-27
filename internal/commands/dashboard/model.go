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
	// eventOverlap re-reads the tail of the last poll. An event is stamped
	// before the transaction that writes it commits, so it can land behind
	// events already read.
	eventOverlap = 30 * time.Second
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
	// StashPoll is how often to read the stash for its feed; zero reads it
	// only with each refresh.
	StashPoll time.Duration
	Now       func() time.Time
	Links     Links
	// Open opens a URL in the browser.
	Open func(url string) error
	// Focus is the environment to center on; empty is the most recently
	// deployed one.
	Focus string
	// Depth is how many hops from the focus to read.
	Depth int
	// Theme is how the map is drawn.
	Theme string
	// Watch takes the view to each alarm that starts firing.
	Watch bool
	// Notify tells someone away from the dashboard that an alarm started firing.
	Notify func(title, body string)
}

type (
	snapshotMsg struct {
		snap *Snapshot
		err  error
		at   time.Time
		// focus is the focus the read was for.
		focus string
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
	stashMsg struct {
		read    StashRead
		initial bool
	}
	frameMsg     time.Time
	refreshMsg   struct{}
	eventPollMsg struct{}
	stashPollMsg struct{}
	refocusMsg   struct {
		id  string
		seq int
	}
	directoryMsg struct {
		places []Place
		err    error
		query  string
	}
)

const (
	// refocusDelay is how long a selection rests before the view moves its
	// focus there, so stepping across the map does not read every stop.
	refocusDelay = 700 * time.Millisecond
	moveFrames   = 14
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
	focus         string
	shownFocus    string
	refocusSeq    int
	directory     []Place
	from          map[string]Point
	moveAt        int
	stash         Stash
	stashOn       bool
	watch         watchState
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
	if cfg.Theme == "" {
		cfg.Theme = DefaultTheme
	}
	return Model{
		api: api, cfg: cfg, loading: true, blastOn: true, showApps: true, introAt: -1, focus: cfg.Focus,
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
	api, ctx, now, where := m.api, m.ctx(), m.cfg.Now, Where{Focus: m.focus, Depth: m.cfg.Depth}
	return func() tea.Msg {
		snap, err := Fetch(ctx, api, where)
		return snapshotMsg{snap: snap, err: err, at: now(), focus: where.Focus}
	}
}

func (m Model) fetchEvents(initial bool) tea.Cmd {
	api, ctx, since := m.api, m.ctx(), m.since.Add(-eventOverlap)
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

func (m Model) fetchStash(initial bool) tea.Cmd {
	if m.snap == nil {
		return nil
	}
	api, ctx, envs := m.api, m.ctx(), m.snap.Environments
	return func() tea.Msg {
		return stashMsg{read: ReadStash(ctx, api, envs), initial: initial}
	}
}

// applyStash takes a new read of the stash: it redraws widgets and notes,
// and past the first read, puts each change on the ticker and flashes where
// it happened.
func (m Model) applyStash(msg stashMsg) (Model, tea.Cmd) {
	var next tea.Cmd
	if m.cfg.StashPoll > 0 {
		next = after(m.cfg.StashPoll, stashPollMsg{})
	}
	if m.graph == nil {
		return m, next
	}
	st := msg.read.Stash
	st.Keep(m.stash, msg.read.Failed)
	var changes []StashChange
	if !msg.initial {
		changes = st.Changes(m.stash)
	}
	m.stash = st
	now := m.cfg.Now()
	m.graph.ApplyStash(m.stash, m.snap.Annotations, now)
	if len(changes) == 0 {
		return m, next
	}
	fresh := make([]Event, 0, len(changes))
	ids := map[string]bool{}
	flashes := make(map[string]Flash, len(m.flashes)+len(changes))
	for id, f := range m.flashes {
		flashes[id] = f
	}
	for _, c := range changes {
		ev := m.graph.StashEvent(c, now)
		fresh = append(fresh, ev)
		ids[ev.ID] = true
		if ev.EnvID != "" && !m.replay.on {
			flashes[ev.EnvID] = Flash{Label: c.Key}
		}
	}
	m.flashes = flashes
	kept := m.events[:0:0]
	for _, ev := range m.events {
		if !ids[ev.ID] {
			kept = append(kept, ev)
		}
	}
	m.events = slices.Concat(fresh, kept)
	m.events = m.events[:min(len(m.events), eventsKept)]
	return m, next
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
	// Only a key moves the focus: a search still being typed, or a selection
	// --watch makes, does not.
	_, key := msg.(tea.KeyMsg)
	if key && !next.searching && next.graph != nil && next.selected != m.selected && next.selected != next.focus {
		next.refocusSeq++
		cmd = tea.Batch(cmd, after(refocusDelay, refocusMsg{id: next.selected, seq: next.refocusSeq}))
	}
	return next, cmd
}

// refocus moves the focus to the selected environment, once the selection
// has rested there, and reads what is within reach of it.
func (m Model) refocus(msg refocusMsg) (Model, tea.Cmd) {
	env := m.graph.Env(msg.id)
	if msg.seq != m.refocusSeq || m.selected != msg.id || msg.id == m.focus || env == nil || env.Restricted {
		return m, nil
	}
	m.focus, m.loading = msg.id, true
	return m, m.fetch()
}

// searchDirectory finds an environment that is not in view by name, and
// moves the focus there.
func (m Model) searchDirectory(query string) (Model, tea.Cmd) {
	if m.directory != nil {
		return m.applyDirectory(directoryMsg{places: m.directory, query: query})
	}
	api, ctx := m.api, m.ctx()
	return m, func() tea.Msg {
		places, err := drain(ctx, api.Directory)
		return directoryMsg{places: places, err: err, query: query}
	}
}

func (m Model) applyDirectory(msg directoryMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.toast, m.toastAt = "could not search: "+clean(msg.err.Error()), m.frame
		return m, nil
	}
	m.directory = msg.places
	p, ok := FindPlace(m.directory, msg.query)
	if !ok {
		m.toast, m.toastAt = "no environment matches "+clean(msg.query), m.frame
		return m, nil
	}
	if p.ID == m.focus {
		return m, nil
	}
	m.focus, m.loading = p.ID, true
	return m, m.fetch()
}

// animatedPos eases each node from where it was before the focus moved to
// where it is now. A node new to the view grows out of the new focus.
func (m Model) animatedPos() map[string]Point {
	t := float64(m.frame-m.moveAt) / moveFrames
	if m.from == nil || t >= 1 {
		return m.pos
	}
	t = 1 - (1-t)*(1-t)*(1-t)
	origin, ok := m.from[m.focus]
	if !ok {
		origin = m.pos[m.focus]
	}
	out := make(map[string]Point, len(m.pos))
	for id, to := range m.pos {
		from, ok := m.from[id]
		if !ok {
			from = origin
		}
		out[id] = Point{X: from.X + (to.X-from.X)*t, Y: from.Y + (to.Y-from.Y)*t}
	}
	return out
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
	case stashPollMsg:
		return m, m.fetchStash(false)
	case stashMsg:
		return m.applyStash(msg)
	case refocusMsg:
		return m.refocus(msg)
	case directoryMsg:
		return m.applyDirectory(msg)
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
	return m.advanceReplay().settleWatch()
}

func (m Model) reload() (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	return m, m.fetch()
}

func (m Model) applySnapshot(msg snapshotMsg) (Model, tea.Cmd) {
	if msg.focus != m.focus {
		return m, nil
	}
	if msg.err != nil && m.graph != nil {
		m.focus = m.shownFocus
	}
	m.loading = false
	m.err = msg.err
	var alert tea.Cmd
	if msg.err == nil {
		before := m.graph
		m.snap, m.graph, m.fetched = msg.snap, BuildGraph(msg.snap), msg.at
		m.layoutKey = ""
		m.focus = msg.snap.Focus
		if m.shownFocus != m.focus {
			if m.shownFocus != "" {
				m.from, m.moveAt = m.pos, m.frame
			}
			m.shownFocus, m.selected = m.focus, m.focus
		}
		if m.introAt < 0 {
			m.introAt = m.frame
		}
		if m.graph.Env(m.selected) == nil {
			m.selected = m.firstSelection()
		}
		m.runtimeIdx = min(m.runtimeIdx, max(len(m.snap.Runtimes)-1, 0))
		m.alarmIdx = min(m.alarmIdx, max(len(m.graph.Firing)-1, 0))
		m.upgradeIdx = min(m.upgradeIdx, max(len(m.upgrades())-1, 0))
		m, alert = m.watchAlarms(before)
		m.graph.ApplyStash(m.stash, m.snap.Annotations, m.cfg.Now())
		if !m.stashOn || m.cfg.StashPoll <= 0 {
			alert = tea.Batch(alert, m.fetchStash(!m.stashOn))
			m.stashOn = true
		}
	}
	return m, tea.Batch(after(m.cfg.Refresh, refreshMsg{}), alert)
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
	for id, at := range m.seen {
		if at.Before(m.since.Add(-eventOverlap)) {
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
		deployed = deployed || m.changesTheView(ev)
	}
	m.flashes = flashes
	if deployed && !m.loading {
		m.loading = true
		return m, tea.Batch(poll, m.fetch())
	}
	return m, poll
}

// changesTheMap reports whether an event changes something the map draws
// that only a full read picks up: a deployment or an alarm. The stash has its
// own feed.
func (m Model) changesTheView(ev Event) bool {
	return changesTheMap(ev.Type) && m.graph != nil && m.graph.Env(ev.EnvID) != nil
}

func changesTheMap(eventType string) bool {
	for _, prefix := range []string{"deployment.", "alarm"} {
		if strings.HasPrefix(eventType, prefix) {
			return true
		}
	}
	return false
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
	if len(m.watch.open) > 0 && msg.String() == "esc" && !m.watch.acked && !m.searching && !m.replay.picking {
		m.watch.acked = true
		return m, nil
	}
	tab, selected := m.tab, m.selected
	next, cmd := m.handleViewKey(msg)
	if next.watch.back != nil && (next.tab != tab || next.selected != selected) {
		next.watch.touched = true
	}
	return next, cmd
}

func (m Model) handleViewKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.replay.picking {
		return m.handlePickKey(msg.String())
	}
	switch key := msg.String(); key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "1", "2", "3", "4":
		m.tab = Tab(key[0] - '1')
	case "enter":
		if m.tab == TabConstellation && m.graph != nil && m.selected != m.focus {
			return m.refocus(refocusMsg{id: m.selected, seq: m.refocusSeq})
		}
		m = m.move(key)
	case "right", "l", "left", "h", "up", "k", "down", "j", "tab", "shift+tab":
		m = m.move(key)
	case "r":
		m.directory = nil
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
	case "v":
		m.cfg.Theme = NextTheme(m.cfg.Theme)
	case "/":
		m.searching, m.filter = true, ""
	}
	return m
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searching = false
		query := m.filter
		next := m.selectFirstMatch()
		next.refocusSeq++
		if next.graph != nil && !(Scene{Filter: query}).Matches(next.graph.Env(next.selected)) {
			next.filter = ""
			return next.searchDirectory(query)
		}
		return next.refocus(refocusMsg{id: next.selected, seq: next.refocusSeq})
	case "esc":
		m.searching, m.filter = false, ""
	case "backspace":
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.filter += string(msg.Runes)
			return m.selectFirstMatch(), nil
		}
	}
	return m, nil
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
			return m.showOnMap(m.graph.Firing[m.alarmIdx].EnvID)
		}
		m.alarmIdx = step(m.alarmIdx, len(m.graph.Firing), key)
	case TabUpgrades:
		ups := m.upgrades()
		if key == "enter" && len(ups) > 0 {
			return m.showOnMap(ups[m.upgradeIdx].EnvID)
		}
		m.upgradeIdx = step(m.upgradeIdx, len(ups), key)
	}
	return m
}

func (m Model) showOnMap(envID string) Model {
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
	key := fmt.Sprintf("%d:%d:%t:%t:%s:%s", w, h, m.grouped, m.stacked, m.cfg.Theme, m.focus)
	if key == m.layoutKey && m.pos != nil {
		return m
	}
	m.pos = LayoutTheme(m.cfg.Theme, m.graph, w, h, LayoutOptions{GroupByProject: m.grouped, ProducersOnTop: m.stacked, Center: m.focus})
	m.layoutKey = key
	return m
}

// Selected is the id of the selected environment.
func (m Model) Selected() string { return m.selected }

// ActiveTab is the tab on screen.
func (m Model) ActiveTab() Tab { return m.tab }

// Focus is the environment the view is centered on.
func (m Model) Focus() string { return m.focus }

// Theme is how the map is drawn.
func (m Model) Theme() string { return m.cfg.Theme }
