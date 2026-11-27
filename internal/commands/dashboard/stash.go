package dashboard

import (
	"context"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// maxStashScopes is the most stashes one read takes.
	maxStashScopes = 200
	stashReads     = 4
)

// Scope names one stash: a project, environment, component, or instance.
type Scope struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// StashEntry is one key in a stash.
type StashEntry struct {
	Scope     Scope  `json:"scope"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	Data      Values `json:"data"`
	Format    string `json:"format"`
	ExpiresAt string `json:"expiresAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Stash is every entry the dashboard reads, by scope and key.
type Stash map[Scope]map[string]StashEntry

// StashChange is a key that was set or deleted between two reads. Entry is
// nil when the key was deleted.
type StashChange struct {
	Scope Scope
	Key   string
	Entry *StashEntry
}

func scopeOf(kind, id string) Scope { return Scope{Kind: strings.ToUpper(kind), ID: id} }

// Get returns the entry for key in scope, unless it has expired.
func (st Stash) Get(scope Scope, key string, now time.Time) (StashEntry, bool) {
	e, ok := st[scopeOf(scope.Kind, scope.ID)][key]
	if !ok || e.expired(now) {
		return StashEntry{}, false
	}
	return e, true
}

// Values are the live entries of a scope by key, the way a note's template
// reads them: a JSON entry as its object, a TEXT entry as its string.
func (st Stash) Values(scope Scope, now time.Time) map[string]any {
	out := map[string]any{}
	for key, e := range st[scopeOf(scope.Kind, scope.ID)] {
		if !e.expired(now) {
			out[key] = e.value()
		}
	}
	return out
}

func (e StashEntry) expired(now time.Time) bool {
	at, err := time.Parse(time.RFC3339, e.ExpiresAt)
	return err == nil && !at.After(now)
}

func (e StashEntry) value() any {
	if e.Format == "JSON" {
		return map[string]any(e.Data)
	}
	return e.Value
}

func (e StashEntry) updated() time.Time {
	at, _ := time.Parse(time.RFC3339, e.UpdatedAt)
	return at
}

// Changes lists the keys set or deleted since prev.
func (st Stash) Changes(prev Stash) []StashChange {
	var out []StashChange
	for scope, entries := range st {
		for key, e := range entries {
			old, ok := prev[scope][key]
			if !ok || old.UpdatedAt != e.UpdatedAt || old.Value != e.Value {
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

const stashQuery = `query dashboardStash($organizationId: ID!, $scopes: [AlphaStashScope!]!, $cursor: Cursor) {
  alphaStashEntriesForScopes(organizationId: $organizationId, scopes: $scopes, cursor: $cursor) {
    cursor { next }
    items { scope { kind id } key label value data format expiresAt updatedAt }
  }
}`

func (a *gqlAPI) StashEntries(ctx context.Context, scopes []Scope, cursor string) ([]StashEntry, string, error) {
	input := make([]map[string]string, 0, len(scopes))
	for _, s := range scopes {
		input = append(input, map[string]string{strings.ToLower(s.Kind): s.ID})
	}
	cur := map[string]any{"limit": pageLimit}
	if cursor != "" {
		cur["next"] = cursor
	}
	vars := map[string]any{"organizationId": a.orgID, "scopes": input, "cursor": cur}
	var data struct {
		Page page[StashEntry] `json:"alphaStashEntriesForScopes"`
	}
	err := a.request(ctx, "dashboardStash", stashQuery, vars, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

// stashScopes groups every project, environment, component, and instance of
// a snapshot by project, in batches a read accepts: every scope of a read
// must be in one project.
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
		add(id, scopeOf("PROJECT", id))
		add(id, scopeOf("ENVIRONMENT", env.ID))
		for _, inst := range env.Instances {
			if inst.Component != nil {
				add(id, scopeOf("COMPONENT", inst.Component.ID))
			}
			add(id, scopeOf("INSTANCE", inst.ID))
		}
	}
	var batches [][]Scope
	for _, id := range projects {
		scopes := byProject[id]
		for len(scopes) > 0 {
			n := min(len(scopes), maxStashScopes)
			batches = append(batches, scopes[:n])
			scopes = scopes[n:]
		}
	}
	return batches
}

// ReadStash reads the stash of every project, environment, component, and
// instance. A batch that fails leaves out only its own entries; the error is
// the last one seen, returned beside what was read.
func ReadStash(ctx context.Context, api API, envs []Environment) (Stash, error) {
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
		scope := scopeOf(e.Scope.Kind, e.Scope.ID)
		if st[scope] == nil {
			st[scope] = map[string]StashEntry{}
		}
		st[scope][e.Key] = e
	}
	var err error
	for _, e := range errs {
		if e != nil {
			err = e
		}
	}
	return st, err
}

func cleanEntry(e StashEntry) StashEntry {
	e.Key, e.Label, e.Value = clean(e.Key), clean(e.Label), clean(e.Value)
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

// envOfScope is the environment a stash scope belongs to, for flashing it.
func (g *Graph) envOfScope(s Scope) string {
	switch s.Kind {
	case "ENVIRONMENT":
		return s.ID
	case "INSTANCE":
		if inst := g.Instance(s.ID); inst != nil {
			return inst.EnvID
		}
	}
	return ""
}

// StashEvent describes a stash change for the ticker.
func (g *Graph) StashEvent(c StashChange, at time.Time) Event {
	ev := Event{ID: "stash:" + c.Scope.Kind + ":" + c.Scope.ID + ":" + c.Key, At: at, EnvID: g.envOfScope(c.Scope)}
	if c.Entry == nil {
		ev.Type, ev.Detail = "stash.deleted", c.Key
		return ev
	}
	ev.Type = "stash.set"
	if t := c.Entry.updated(); !t.IsZero() {
		ev.At = t
	}
	ev.Detail = c.Key + " = " + preview(c.Entry)
	if ev.EnvID == "" {
		ev.Detail = strings.ToLower(c.Scope.Kind) + " " + c.Scope.ID + " · " + ev.Detail
	}
	return ev
}

func preview(e *StashEntry) string {
	v := strings.Join(strings.Fields(e.Value), " ")
	if len([]rune(v)) > 60 {
		v = string([]rune(v)[:59]) + "…"
	}
	return v
}
