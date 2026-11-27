package dashboard

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
	"golang.org/x/sync/errgroup"
)

const stashReads = 4

// Scope names one stash: a project, environment, component, or instance.
type Scope = stash.Scope

// StashEntry is one key in a stash.
type StashEntry = stash.Entry

// StashEvent says a key in a stash was set or deleted.
type StashEvent = stash.Event

// Stash is every entry the dashboard reads, by scope and key.
type Stash map[Scope]map[string]StashEntry

// StashChange is a key that was set or deleted between two reads. Entry is
// nil when the key was deleted.
type StashChange struct {
	Scope Scope
	Key   string
	Entry *StashEntry
}

// scopeOf names a scope from a kind spelled in any case, as a widget's
// inputs spell it.
func scopeOf(kind stash.ScopeKind, id string) Scope {
	return Scope{Kind: stash.ScopeKind(strings.ToLower(string(kind))), ID: id}
}

// Get returns the entry for key in scope, unless it has expired.
func (st Stash) Get(scope Scope, key string, now time.Time) (StashEntry, bool) {
	e, ok := st[scopeOf(scope.Kind, scope.ID)][key]
	if !ok || expired(e, now) {
		return StashEntry{}, false
	}
	return e, true
}

// Values are the live entries of a scope by key, the way a note's template
// reads them: a JSON entry as its object, a TEXT entry as its string.
func (st Stash) Values(scope Scope, now time.Time) map[string]any {
	out := map[string]any{}
	for key, e := range st[scopeOf(scope.Kind, scope.ID)] {
		if !expired(e, now) {
			out[key] = valueOf(e)
		}
	}
	return out
}

func expired(e StashEntry, now time.Time) bool {
	return !e.ExpiresAt.IsZero() && !e.ExpiresAt.After(now)
}

func valueOf(e StashEntry) any {
	if e.Format == string(stash.FormatJSON) {
		return e.Data
	}
	return text(e)
}

func text(e StashEntry) string {
	if e.Value == nil {
		return ""
	}
	return *e.Value
}

// Apply returns the stash with events applied, leaving st as it was.
func (st Stash) Apply(events []StashEvent) Stash {
	out := maps.Clone(st)
	if out == nil {
		out = Stash{}
	}
	copied := map[Scope]bool{}
	for _, ev := range events {
		if !copied[ev.Scope] {
			out[ev.Scope] = maps.Clone(st[ev.Scope])
			if out[ev.Scope] == nil {
				out[ev.Scope] = map[string]StashEntry{}
			}
			copied[ev.Scope] = true
		}
		switch {
		case ev.Action == stash.EventDeleted:
			delete(out[ev.Scope], ev.Key)
		case ev.Entry != nil:
			e := cleanEntry(*ev.Entry)
			e.Scope = ev.Scope
			out[ev.Scope][ev.Key] = e
		}
	}
	return out
}

// KeepTouched copies from prev each key that touched says changed after
// since: a read that started before an event arrived would undo it.
func (st Stash) KeepTouched(prev Stash, touched map[Scope]map[string]time.Time, since time.Time) {
	for scope, keys := range touched {
		for key, at := range keys {
			if !at.After(since) {
				continue
			}
			e, ok := prev[scope][key]
			switch {
			case ok && st[scope] == nil:
				st[scope] = map[string]StashEntry{key: e}
			case ok:
				st[scope][key] = e
			default:
				delete(st[scope], key)
			}
		}
	}
}

// Changes lists the keys set or deleted since prev.
func (st Stash) Changes(prev Stash) []StashChange {
	var out []StashChange
	for scope, entries := range st {
		for key, e := range entries {
			old, ok := prev[scope][key]
			if !ok || !old.UpdatedAt.Equal(e.UpdatedAt) || text(old) != text(e) || !reflect.DeepEqual(old.Data, e.Data) {
				out = append(out, StashChange{Scope: scope, Key: key, Entry: &e})
			}
		}
	}
	for scope, entries := range prev {
		for key := range entries {
			if _, ok := st[scope][key]; !ok {
				out = append(out, StashChange{Scope: scope, Key: key})
			}
		}
	}
	slices.SortFunc(out, func(a, b StashChange) int {
		return strings.Compare(a.Scope.ID+"\x00"+a.Key, b.Scope.ID+"\x00"+b.Key)
	})
	return out
}

// StashService reads the stash of many scopes at once, and streams its
// changes.
type StashService interface {
	ListForScopes(ctx context.Context, scopes []stash.Scope, input stash.ForScopesInput) (types.Page[stash.Entry], error)
	Subscribe(ctx context.Context, projectID, environmentID string) (*streaming.Stream[stash.Event], error)
}

func (a *gqlAPI) StashEntries(ctx context.Context, scopes []Scope, cursor string) ([]StashEntry, string, error) {
	page, err := a.stash.ListForScopes(ctx, scopes, stash.ForScopesInput{PageSize: pageLimit, After: cursor})
	return page.Items, page.Next, err
}

func (a *gqlAPI) SubscribeStash(ctx context.Context, projectID, environmentID string) (*streaming.Stream[StashEvent], error) {
	return a.stash.Subscribe(ctx, projectID, environmentID)
}

// Every scope of a read must be in one project.
func stashScopes(envs []Environment) [][]Scope {
	byProject := map[string][]Scope{}
	seen := map[Scope]bool{}
	var projects []string
	add := func(project string, s Scope) {
		if !seen[s] {
			seen[s] = true
			byProject[project] = append(byProject[project], s)
		}
	}
	for _, env := range envs {
		if env.Project == nil {
			continue
		}
		id := env.Project.ID
		if _, ok := byProject[id]; !ok {
			projects = append(projects, id)
		}
		add(id, Scope{Kind: stash.ScopeProject, ID: id})
		add(id, Scope{Kind: stash.ScopeEnvironment, ID: env.ID})
		for _, inst := range env.Instances {
			if inst.Component != nil {
				add(id, Scope{Kind: stash.ScopeComponent, ID: inst.Component.ID})
			}
			add(id, Scope{Kind: stash.ScopeInstance, ID: inst.ID})
		}
	}
	var batches [][]Scope
	for _, id := range projects {
		scopes := byProject[id]
		for len(scopes) > 0 {
			n := min(len(scopes), stash.MaxScopes)
			batches = append(batches, scopes[:n])
			scopes = scopes[n:]
		}
	}
	return batches
}

// StashRead is one read of the stash. Failed holds the scopes of the
// batches that could not be read, whose entries are unknown rather than gone.
type StashRead struct {
	Stash Stash
	// Scopes are every scope the read asked for, read or not.
	Scopes []Scope
	Failed []Scope
	Err    error
}

// ReadStash reads the stash of every project, environment, component, and
// instance.
func ReadStash(ctx context.Context, api API, envs []Environment) StashRead {
	batches := stashScopes(envs)
	results := make([][]StashEntry, len(batches))
	errs := make([]error, len(batches))
	var group errgroup.Group
	group.SetLimit(stashReads)
	for i, scopes := range batches {
		group.Go(func() error {
			page := func(ctx context.Context, cursor string) ([]StashEntry, string, error) {
				return api.StashEntries(ctx, scopes, cursor)
			}
			results[i], errs[i] = drain(ctx, page)
			return nil
		})
	}
	_ = group.Wait()
	st := Stash{}
	for _, e := range slices.Concat(results...) {
		e = cleanEntry(e)
		if st[e.Scope] == nil {
			st[e.Scope] = map[string]StashEntry{}
		}
		st[e.Scope][e.Key] = e
	}
	read := StashRead{Stash: st, Scopes: slices.Concat(batches...)}
	for i, e := range errs {
		if e != nil {
			read.Err = e
			read.Failed = append(read.Failed, batches[i]...)
		}
	}
	return read
}

// Keep copies the entries of scopes from prev, for scopes a read could not
// reach.
func (st Stash) Keep(prev Stash, scopes []Scope) {
	for _, s := range scopes {
		if entries, ok := prev[s]; ok {
			st[s] = entries
		}
	}
}

func cleanEntry(e StashEntry) StashEntry {
	e.Key, e.Label = clean(e.Key), clean(e.Label)
	if e.Value != nil {
		v := clean(*e.Value)
		e.Value = &v
	}
	e.Data = cleanValues(e.Data)
	return e
}

func cleanValues(v Values) Values {
	if v == nil {
		return nil
	}
	out := make(Values, len(v))
	for k, val := range v {
		out[clean(k)] = cleanAny(val)
	}
	return out
}

func cleanAny(v any) any {
	switch t := v.(type) {
	case string:
		return clean(t)
	case map[string]any:
		return map[string]any(cleanValues(t))
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = cleanAny(item)
		}
		return out
	default:
		return v
	}
}

func (g *Graph) envOfScope(s Scope) string {
	switch s.Kind {
	case stash.ScopeEnvironment:
		return s.ID
	case stash.ScopeInstance:
		if inst := g.Instance(s.ID); inst != nil {
			return inst.EnvID
		}
	case stash.ScopeProject, stash.ScopeComponent, stash.ScopeDeployment:
	}
	return ""
}

// StashEvent describes a stash change for the ticker.
func (g *Graph) StashEvent(c StashChange, at time.Time) Event {
	ev := Event{ID: "stash:" + c.Scope.String() + ":" + c.Key, At: at, EnvID: g.envOfScope(c.Scope)}
	if c.Entry == nil {
		ev.Type, ev.Detail = "stash.deleted", c.Key
		return ev
	}
	ev.Type = "stash.set"
	if !c.Entry.UpdatedAt.IsZero() {
		ev.At = c.Entry.UpdatedAt
	}
	ev.Detail = c.Key + " = " + preview(c.Entry)
	if ev.EnvID == "" {
		ev.Detail = string(c.Scope.Kind) + " " + c.Scope.ID + " · " + ev.Detail
	}
	return ev
}

func preview(e *StashEntry) string {
	v := text(*e)
	if v == "" && e.Data != nil {
		data, _ := json.Marshal(e.Data)
		v = string(data)
	}
	v = strings.Join(strings.Fields(v), " ")
	if len([]rune(v)) > 60 {
		v = string([]rune(v)[:59]) + "…"
	}
	return v
}
