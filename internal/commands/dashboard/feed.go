package dashboard

import (
	"context"
	"errors"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
)

// The stash feed reads every stash in view once, then streams each key as it
// is set or deleted, one subscription for each environment in view. When a
// profile cannot stream, or while a subscription is down, it polls instead.

const (
	// eventBatch is the most stash events applied in one update, so a burst
	// redraws once rather than once an event.
	eventBatch = 64
	// touchedFor is how long a streamed key is remembered, to keep a read
	// that started before the event from undoing it.
	touchedFor = time.Minute
	maxBackoff = time.Minute
	// maxStreams is the most environments streamed at once. A view wider
	// than this polls instead of holding a stream open for each.
	maxStreams = 20
)

type (
	subscribedMsg struct {
		env    string
		gen    int
		stream *streaming.Stream[StashEvent]
		err    error
	}
	stashEventsMsg struct {
		env    string
		gen    int
		events []StashEvent
	}
	stashClosedMsg struct {
		env string
		gen int
		err error
	}
	resubscribeMsg struct {
		env string
		gen int
	}
)

// subscription is one environment's stream. gen tells its messages from
// those of an earlier stream for the same environment.
type subscription struct {
	project  string
	gen      int
	cancel   context.CancelFunc
	stream   *streaming.Stream[StashEvent]
	failures int
}

// feed is the state of the stash feed. Its maps are shared between copies
// of the model, like the rest of its bookkeeping, and only Update writes them.
type feed struct {
	subs map[string]*subscription
	gen  int
	// cannot is set when the profile cannot stream at all.
	cannot  bool
	reading bool
	// pending is a read asked for while one was in flight, run when it lands
	// so environments new to the view get their first read.
	pending bool
	touched map[Scope]map[string]time.Time
	// known holds the scopes the latest read covered. A scope's first read
	// is what it already had, not news for the ticker.
	known map[Scope]bool
}

// live reports whether every environment in view is streaming, so polling
// would only repeat what the streams say.
func (m Model) live() bool {
	if m.feed.cannot || len(m.feed.subs) == 0 || m.snap == nil {
		return false
	}
	for _, env := range m.snap.Environments {
		if _, ok := m.feed.subs[env.ID]; !ok && env.Project != nil {
			return false
		}
	}
	for _, sub := range m.feed.subs {
		if sub.stream == nil {
			return false
		}
	}
	return true
}

func (m Model) fetchStash(initial bool) (Model, tea.Cmd) {
	if m.snap == nil {
		return m, nil
	}
	if m.feed.reading {
		m.feed.pending = true
		return m, nil
	}
	m.feed.reading = true
	api, ctx, envs, now := m.api, m.ctx(), m.snap.Environments, m.cfg.Now
	return m, func() tea.Msg {
		at := now()
		return stashMsg{read: ReadStash(ctx, api, envs), initial: initial, at: at}
	}
}

func (m Model) pollStash() (Model, tea.Cmd) {
	next := after(m.cfg.StashPoll, stashPollMsg{})
	if m.live() {
		return m, next
	}
	m, read := m.fetchStash(false)
	return m, tea.Batch(read, next)
}

func (m Model) applyStash(msg stashMsg) (Model, tea.Cmd) {
	m.feed.reading = false
	var again tea.Cmd
	if m.feed.pending {
		m.feed.pending = false
		m, again = m.fetchStash(false)
	}
	if m.graph == nil {
		return m, again
	}
	st := msg.read.Stash
	st.Keep(m.stash, msg.read.Failed)
	st.KeepTouched(m.stash, m.feed.touched, msg.at)
	var news func(Scope) bool
	if prev := m.feed.known; !msg.initial {
		news = func(s Scope) bool { return prev[s] }
	}
	failed := map[Scope]bool{}
	for _, s := range msg.read.Failed {
		failed[s] = true
	}
	known := make(map[Scope]bool, len(msg.read.Scopes))
	for _, s := range msg.read.Scopes {
		if !failed[s] || m.feed.known[s] {
			known[s] = true
		}
	}
	m.feed.known = known
	return m.takeStash(st, news), again
}

func (m Model) takeStash(st Stash, news func(Scope) bool) Model {
	var changes []StashChange
	if news != nil {
		for _, c := range st.Changes(m.stash) {
			if news(c.Scope) {
				changes = append(changes, c)
			}
		}
	}
	m.stash = st
	now := m.cfg.Now()
	m.graph.ApplyStash(m.stash, m.snap.Annotations, now)
	if len(changes) == 0 {
		return m
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
	return m
}

func (m Model) syncSubscriptions() (Model, tea.Cmd) {
	if m.feed.cannot || m.snap == nil {
		return m, nil
	}
	if m.feed.subs == nil {
		m.feed.subs = map[string]*subscription{}
	}
	want := map[string]string{}
	for _, env := range m.snap.Environments {
		if env.Project != nil {
			want[env.ID] = env.Project.ID
		}
	}
	if len(want) > maxStreams {
		clear(want)
	}
	for id, sub := range m.feed.subs {
		if _, ok := want[id]; !ok {
			sub.cancel()
			delete(m.feed.subs, id)
		}
	}
	var cmds []tea.Cmd
	for id, project := range want {
		if _, ok := m.feed.subs[id]; ok {
			continue
		}
		var cmd tea.Cmd
		m, cmd = m.subscribe(id, project, 0)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) subscribe(env, project string, failures int) (Model, tea.Cmd) {
	if old, ok := m.feed.subs[env]; ok {
		old.cancel()
	}
	m.feed.gen++
	ctx, cancel := context.WithCancel(m.ctx())
	sub := &subscription{project: project, gen: m.feed.gen, cancel: cancel, failures: failures}
	m.feed.subs[env] = sub
	api, gen := m.api, sub.gen
	return m, func() tea.Msg {
		stream, err := api.SubscribeStash(ctx, project, env)
		return subscribedMsg{env: env, gen: gen, stream: stream, err: err}
	}
}

func (m Model) current(env string, gen int) *subscription {
	if sub, ok := m.feed.subs[env]; ok && sub.gen == gen {
		return sub
	}
	return nil
}

func (m Model) applySubscribed(msg subscribedMsg) (Model, tea.Cmd) {
	sub := m.current(msg.env, msg.gen)
	switch {
	case sub == nil:
		return m, nil
	case errors.Is(msg.err, streaming.ErrRequiresPAT):
		m.feed.cannot = true
		for _, s := range m.feed.subs {
			s.cancel()
		}
		m.feed.subs = nil
		return m, nil
	case msg.err != nil:
		return m.retry(msg.env, sub)
	}
	sub.stream = msg.stream
	cmds := []tea.Cmd{receive(msg.env, msg.gen, msg.stream)}
	if sub.failures > 0 {
		// The stream was down for a while; read what it missed.
		var read tea.Cmd
		m, read = m.fetchStash(false)
		cmds = append(cmds, read)
	}
	return m, tea.Batch(cmds...)
}

// receive takes every event already waiting along with the first, so a
// burst redraws once.
func receive(env string, gen int, stream *streaming.Stream[StashEvent]) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-stream.C
		if !ok {
			return stashClosedMsg{env: env, gen: gen, err: stream.Err()}
		}
		events := []StashEvent{ev}
		for len(events) < eventBatch {
			select {
			case ev, ok := <-stream.C:
				if !ok {
					return stashEventsMsg{env: env, gen: gen, events: events}
				}
				events = append(events, ev)
			default:
				return stashEventsMsg{env: env, gen: gen, events: events}
			}
		}
		return stashEventsMsg{env: env, gen: gen, events: events}
	}
}

func (m Model) applyStashEvents(msg stashEventsMsg) (Model, tea.Cmd) {
	sub := m.current(msg.env, msg.gen)
	if sub == nil {
		return m, nil
	}
	sub.failures = 0
	now := m.cfg.Now()
	if m.feed.touched == nil {
		m.feed.touched = map[Scope]map[string]time.Time{}
	}
	for scope, keys := range m.feed.touched {
		for key, at := range keys {
			if now.Sub(at) > touchedFor {
				delete(keys, key)
			}
		}
		if len(keys) == 0 {
			delete(m.feed.touched, scope)
		}
	}
	for _, ev := range msg.events {
		if m.feed.touched[ev.Scope] == nil {
			m.feed.touched[ev.Scope] = map[string]time.Time{}
		}
		m.feed.touched[ev.Scope][ev.Key] = now
	}
	if m.graph != nil {
		m = m.takeStash(m.stash.Apply(msg.events), func(Scope) bool { return true })
	}
	return m, receive(msg.env, msg.gen, sub.stream)
}

func (m Model) applyStashClosed(msg stashClosedMsg) (Model, tea.Cmd) {
	sub := m.current(msg.env, msg.gen)
	if sub == nil {
		return m, nil
	}
	return m.retry(msg.env, sub)
}

func (m Model) retry(env string, sub *subscription) (Model, tea.Cmd) {
	sub.stream = nil
	sub.failures++
	if sub.failures == 1 {
		return m.subscribe(env, sub.project, sub.failures)
	}
	wait := min(time.Second<<(sub.failures-2), maxBackoff)
	return m, after(wait, resubscribeMsg{env: env, gen: sub.gen})
}

func (m Model) applyResubscribe(msg resubscribeMsg) (Model, tea.Cmd) {
	sub := m.current(msg.env, msg.gen)
	if sub == nil {
		return m, nil
	}
	return m.subscribe(msg.env, sub.project, sub.failures)
}
