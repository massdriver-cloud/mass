package dashboard

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
)

// Widget names in the platform's widget catalog.
const (
	WidgetHealth  = "instance.status_dot"
	WidgetUsage   = "instance.usage_bar"
	WidgetTraffic = "link.traffic"
	WidgetBanner  = "environment.banner"
	WidgetLinks   = "environment.header_link"
)

const (
	// maxWidgetScopes is the most stashes one widgets read takes.
	maxWidgetScopes = 200
	widgetReads     = 4
)

// Scope names one stash: an environment or an instance.
type Scope struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// Widget is one widget drawn from a well-known md/ stash key.
type Widget struct {
	Name   string `json:"name"`
	Scope  Scope  `json:"scope"`
	Values Values `json:"values"`
}

// Values are a widget's inputs by name.
type Values map[string]any

// UnmarshalJSON reads a Map scalar, which may arrive as an object or as a
// JSON-encoded string.
func (v *Values) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err == nil {
		data = []byte(encoded)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*v = m
	return nil
}

func str(v Values, name string) string {
	s, _ := v[name].(string)
	return s
}

func num(v Values, name string) (float64, bool) {
	n, ok := v[name].(float64)
	return n, ok
}

// Usage is how much of a limit an instance uses.
type Usage struct {
	Value, Max float64
	Label      string
}

// Traffic is the traffic on the links that leave an instance.
type Traffic struct {
	Rate   float64
	Errors int
}

// Banner is a message across the top of an environment.
type Banner struct {
	Message string
	Level   string
}

// HeaderLink is a link in an environment's header, like a dashboard or a runbook.
type HeaderLink struct {
	Label string
	URL   string
}

const widgetsQuery = `query dashboardWidgets($organizationId: ID!, $scopes: [AlphaStashScope!]!, $cursor: Cursor) {
  alphaWidgetsForScopes(organizationId: $organizationId, scopes: $scopes, cursor: $cursor) {
    cursor { next }
    items { name scope { kind id } values }
  }
}`

func (a *gqlAPI) Widgets(ctx context.Context, scopes []Scope, cursor string) ([]Widget, string, error) {
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
		Page page[Widget] `json:"alphaWidgetsForScopes"`
	}
	err := a.request(ctx, "dashboardWidgets", widgetsQuery, vars, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

// widgetScopes groups the environments and instances of a snapshot by
// project, in batches the widgets read accepts: every scope of a read must be
// in one project.
func widgetScopes(envs []Environment) [][]Scope {
	byProject := map[string][]Scope{}
	var projects []string
	for _, env := range envs {
		if env.Project == nil {
			continue
		}
		id := env.Project.ID
		if _, ok := byProject[id]; !ok {
			projects = append(projects, id)
		}
		byProject[id] = append(byProject[id], Scope{Kind: "ENVIRONMENT", ID: env.ID})
		for _, inst := range env.Instances {
			byProject[id] = append(byProject[id], Scope{Kind: "INSTANCE", ID: inst.ID})
		}
	}
	var batches [][]Scope
	for _, id := range projects {
		scopes := byProject[id]
		for len(scopes) > 0 {
			n := min(len(scopes), maxWidgetScopes)
			batches = append(batches, scopes[:n])
			scopes = scopes[n:]
		}
	}
	return batches
}

// fetchWidgets reads the widgets of every environment and instance. Widgets
// are extra: a read that fails, as it does on a server without widgets,
// leaves them out rather than failing the dashboard.
func fetchWidgets(ctx context.Context, api API, envs []Environment) []Widget {
	batches := widgetScopes(envs)
	results := make([][]Widget, len(batches))
	var group errgroup.Group
	group.SetLimit(widgetReads)
	for i, scopes := range batches {
		group.Go(func() error {
			page := func(ctx context.Context, cursor string) ([]Widget, string, error) {
				return api.Widgets(ctx, scopes, cursor)
			}
			results[i], _ = drain(ctx, page)
			return nil
		})
	}
	_ = group.Wait()
	return slices.Concat(results...)
}

func (g *Graph) applyWidgets(widgets []Widget) {
	for _, w := range widgets {
		switch w.Scope.Kind {
		case "INSTANCE":
			if inst := g.instances[w.Scope.ID]; inst != nil {
				applyInstanceWidget(inst, w)
			}
		case "ENVIRONMENT":
			if env := g.envs[w.Scope.ID]; env != nil {
				applyEnvWidget(env, w)
			}
		}
	}
}

func applyInstanceWidget(inst *InstNode, w Widget) {
	switch w.Name {
	case WidgetHealth:
		inst.Health = clean(str(w.Values, "state"))
	case WidgetUsage:
		value, okValue := num(w.Values, "value")
		limit, okMax := num(w.Values, "max")
		if okValue && okMax && limit > 0 {
			inst.Usage = &Usage{Value: value, Max: limit, Label: clean(str(w.Values, "label"))}
		}
	case WidgetTraffic:
		rate, ok := num(w.Values, "rate")
		errs, _ := num(w.Values, "errors")
		if ok {
			inst.Traffic = &Traffic{Rate: rate, Errors: int(errs)}
		}
	}
}

func applyEnvWidget(env *EnvNode, w Widget) {
	switch w.Name {
	case WidgetBanner:
		if msg := clean(str(w.Values, "message")); msg != "" {
			env.Banner = &Banner{Message: msg, Level: str(w.Values, "level")}
		}
	case WidgetLinks:
		links, _ := w.Values["links"].([]any)
		for _, l := range links {
			link, _ := l.(map[string]any)
			label, _ := link["label"].(string)
			url, _ := link["url"].(string)
			label, url = clean(label), clean(url)
			if label != "" && webURL(url) {
				env.Links = append(env.Links, HeaderLink{Label: label, URL: url})
			}
		}
	}
}

// webURL keeps a link the terminal can open to http and https.
func webURL(url string) bool {
	for _, scheme := range []string{"https://", "http://"} {
		if len(url) >= len(scheme) && strings.EqualFold(url[:len(scheme)], scheme) {
			return true
		}
	}
	return false
}

// clean strips control characters, so text from the platform cannot end or
// start terminal escape sequences, and bidirectional overrides, so it reads
// on screen the way it is written.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, r >= 0x7f && r < 0xa0:
			return -1
		case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
			return -1
		case r == 0x200e, r == 0x200f, r == 0x061c, r == 0x2028, r == 0x2029:
			return -1
		}
		return r
	}, s)
}

// Traffic sums the traffic that instances in the link's producing
// environment send on lines to its consumer.
func (g *Graph) Traffic(l Link) *Traffic {
	if l.Kind != LinkDependency {
		return nil
	}
	var total *Traffic
	senders := map[string]bool{}
	for _, il := range g.InstLinks {
		from, to := g.instances[il.From], g.instances[il.To]
		if from == nil || to == nil || from.EnvID != l.From || to.EnvID != l.To || from.Traffic == nil || senders[from.ID] {
			continue
		}
		senders[from.ID] = true
		if total == nil {
			total = &Traffic{}
		}
		total.Rate += from.Traffic.Rate
		total.Errors += from.Traffic.Errors
	}
	return total
}
