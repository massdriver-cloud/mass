package dashboard

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Widgets the Massdriver UI draws, named by the surface they show on.
const (
	WidgetHealth  = "instance.status_dot"
	WidgetUsage   = "instance.usage_bar"
	WidgetTraffic = "link.traffic"
	WidgetBanner  = "environment.banner"
	WidgetLinks   = "environment.header_link"
)

// widgetInput is one input a widget reads.
type widgetInput struct {
	name     string
	kind     string
	required bool
	// field is the field of the well-known key's JSON value the input reads;
	// empty for a TEXT key, whose whole value is the input.
	field string
}

// widgetSpec is a widget and the well-known md/ key that draws it without
// an annotation.
type widgetSpec struct {
	key    string
	scope  string
	format string
	inputs []widgetInput
}

var widgetSpecs = map[string]widgetSpec{
	WidgetHealth: {key: "md/health", scope: "INSTANCE", format: "TEXT", inputs: []widgetInput{
		{name: "state", kind: "string", required: true},
	}},
	WidgetUsage: {key: "md/usage", scope: "INSTANCE", format: "JSON", inputs: []widgetInput{
		{name: "value", kind: "number", required: true, field: "value"},
		{name: "max", kind: "number", required: true, field: "max"},
		{name: "label", kind: "string", field: "label"},
	}},
	WidgetTraffic: {key: "md/traffic", scope: "INSTANCE", format: "JSON", inputs: []widgetInput{
		{name: "rate", kind: "number", required: true, field: "rps"},
		{name: "errors", kind: "integer", required: true, field: "errors_5m"},
		{name: "latency", kind: "number", field: "p95_ms"},
	}},
	WidgetBanner: {key: "md/banner", scope: "ENVIRONMENT", format: "JSON", inputs: []widgetInput{
		{name: "message", kind: "string", required: true, field: "message"},
		{name: "level", kind: "string", field: "level"},
	}},
	WidgetLinks: {key: "md/links", scope: "ENVIRONMENT", format: "JSON", inputs: []widgetInput{
		{name: "links", kind: "array", required: true, field: "links"},
	}},
}

// resolved is a widget's input values and when its oldest input was set.
type resolved struct {
	values  map[string]any
	updated time.Time
}

// automatic reads a widget from its well-known key in scope.
func automatic(spec widgetSpec, st Stash, scope Scope, now time.Time) (resolved, bool) {
	e, ok := st.Get(scope, spec.key, now)
	if !ok || e.Format != spec.format {
		return resolved{}, false
	}
	sources := map[string]any{}
	for _, in := range spec.inputs {
		var path []any
		if in.field != "" {
			path = []any{in.field}
		}
		sources[in.name] = map[string]any{"scope": map[string]any{strings.ToLower(scope.Kind): scope.ID}, "key": spec.key, "path": path}
	}
	return resolve(spec, sources, st, now)
}

// resolve reads each input of a widget from the stash entry its source
// names, and fails when a required input is missing or the wrong type.
func resolve(spec widgetSpec, sources map[string]any, st Stash, now time.Time) (resolved, bool) {
	r := resolved{values: map[string]any{}}
	for _, in := range spec.inputs {
		src, _ := sources[in.name].(map[string]any)
		value, updated, ok := lookup(src, st, now)
		if ok {
			value, ok = check(in.kind, value)
		}
		if !ok {
			if in.required {
				return resolved{}, false
			}
			continue
		}
		r.values[in.name] = value
		if r.updated.IsZero() || updated.Before(r.updated) {
			r.updated = updated
		}
	}
	return r, true
}

func lookup(src map[string]any, st Stash, now time.Time) (any, time.Time, bool) {
	key, _ := src["key"].(string)
	scopeMap, _ := src["scope"].(map[string]any)
	if key == "" || len(scopeMap) != 1 {
		return nil, time.Time{}, false
	}
	var scope Scope
	for kind, id := range scopeMap {
		s, _ := id.(string)
		scope = scopeOf(kind, s)
	}
	e, ok := st.Get(scope, key, now)
	if !ok {
		return nil, time.Time{}, false
	}
	path, _ := src["path"].([]any)
	if len(path) > 0 && e.Format != "JSON" {
		return nil, time.Time{}, false
	}
	value := e.value()
	for _, seg := range path {
		value, ok = descend(value, seg)
		if !ok {
			return nil, time.Time{}, false
		}
	}
	return value, e.updated(), true
}

func descend(value, seg any) (any, bool) {
	switch v := value.(type) {
	case map[string]any:
		key, _ := seg.(string)
		out, ok := v[key]
		return out, ok
	case []any:
		i, ok := seg.(float64)
		if !ok || i < 0 || int(i) >= len(v) {
			return nil, false
		}
		return v[int(i)], true
	}
	return nil, false
}

// check converts a value to an input's type. Numbers may arrive as plain
// decimal TEXT, the way a script writes them.
func check(kind string, value any) (any, bool) {
	switch kind {
	case "number", "integer":
		n, ok := value.(float64)
		if s, isText := value.(string); isText {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			n, ok = parsed, err == nil
		}
		if !ok || (kind == "integer" && n != float64(int64(n))) {
			return nil, false
		}
		return n, true
	case "string":
		s, ok := value.(string)
		return s, ok
	case "array":
		a, ok := value.([]any)
		return a, ok
	}
	return nil, false
}

// ApplyStash draws widgets and notes from the stash: a widget an annotation
// pins to an entity, or else the one its well-known md/ key draws, and every
// note annotation rendered against the stashes it can see.
func (g *Graph) ApplyStash(st Stash, annotations []Annotation, now time.Time) {
	g.clearStash()
	pinned := g.applyAnnotations(st, annotations, now)
	for _, env := range g.Envs {
		g.applyAutomatic(env, st, pinned, now)
	}
}

func (g *Graph) applyAnnotations(st Stash, annotations []Annotation, now time.Time) map[string]bool {
	pinned := map[string]bool{}
	for _, a := range annotations {
		if a.Kind == "NOTE" {
			g.applyNote(a, st, now)
			continue
		}
		if a.Kind != "WIDGET" {
			continue
		}
		for _, surface := range a.Surfaces {
			for _, target := range g.applyPinned(a, surface, st, now) {
				pinned[surface+"\x00"+target] = true
			}
		}
	}
	return pinned
}

func (g *Graph) applyAutomatic(env *EnvNode, st Stash, pinned map[string]bool, now time.Time) {
	for name, spec := range widgetSpecs {
		if spec.scope == "ENVIRONMENT" {
			if r, ok := automatic(spec, st, scopeOf("ENVIRONMENT", env.ID), now); ok && !pinned[name+"\x00"+env.ID] {
				applyEnvWidget(env, name, r)
			}
			continue
		}
		for _, inst := range env.Instances {
			if r, ok := automatic(spec, st, scopeOf("INSTANCE", inst.ID), now); ok && !pinned[name+"\x00"+inst.ID] {
				applyInstanceWidget(inst, name, r)
			}
		}
	}
}

func (g *Graph) clearStash() {
	for _, env := range g.Envs {
		env.Banner, env.Links, env.Notes, env.ProjectNotes, env.LinkNotes = nil, nil, nil, nil, nil
		for _, inst := range env.Instances {
			inst.Health, inst.Usage, inst.Traffic, inst.Notes = "", nil, nil, nil
		}
	}
}

func (g *Graph) applyPinned(a Annotation, surface string, st Stash, now time.Time) []string {
	spec, ok := widgetSpecs[surface]
	if !ok {
		return nil
	}
	r, ok := resolve(spec, map[string]any(a.Inputs), st, now)
	if !ok {
		return nil
	}
	var drawn []string
	switch surface {
	case WidgetBanner, WidgetLinks:
		if env := g.Env(a.Entity.ID); env != nil && a.Entity.Typename == "Environment" {
			applyEnvWidget(env, surface, r)
			drawn = append(drawn, env.ID)
		}
	case WidgetTraffic:
		for _, pair := range g.annotatedLinks(a) {
			applyInstanceWidget(pair[0], surface, r)
			drawn = append(drawn, pair[0].ID)
		}
	default:
		for _, inst := range g.annotatedInstances(a) {
			applyInstanceWidget(inst, surface, r)
			drawn = append(drawn, inst.ID)
		}
	}
	return drawn
}

func applyInstanceWidget(inst *InstNode, name string, r resolved) {
	v := Values(r.values)
	switch name {
	case WidgetHealth:
		inst.Health = clean(str(v, "state"))
	case WidgetUsage:
		value, _ := num(v, "value")
		limit, _ := num(v, "max")
		if limit > 0 {
			inst.Usage = &Usage{Value: value, Max: limit, Label: clean(str(v, "label"))}
		}
	case WidgetTraffic:
		rate, _ := num(v, "rate")
		errs, _ := num(v, "errors")
		latency, hasLatency := num(v, "latency")
		inst.Traffic = &Traffic{Rate: rate, Errors: int(errs), Latency: latency, HasLatency: hasLatency, Updated: r.updated}
	}
}

func applyEnvWidget(env *EnvNode, name string, r resolved) {
	v := Values(r.values)
	switch name {
	case WidgetBanner:
		if msg := clean(str(v, "message")); msg != "" {
			env.Banner = &Banner{Message: msg, Level: str(v, "level")}
		}
	case WidgetLinks:
		links, _ := v["links"].([]any)
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

// LinkNote is a line of a note on the link between two instances.
type LinkNote struct {
	From, To string
	Text     string
}

func latencyText(t *Traffic) string {
	if !t.HasLatency {
		return ""
	}
	return fmt.Sprintf(" %.0fms", t.Latency)
}

// Values are a widget's inputs by name.
type Values map[string]any

// UnmarshalJSON reads a Map scalar, which may arrive as an object or as a
// JSON-encoded string.
func (v *Values) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = nil
		return nil
	}
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
	Rate    float64
	Errors  int
	Latency float64
	// HasLatency says Latency was set; zero is a real latency.
	HasLatency bool
	Updated    time.Time
}

// trafficStaleAfter is how long traffic can go unreported before the
// dashboard treats it as stale.
const trafficStaleAfter = 10 * time.Minute

// Stale reports whether the traffic has gone unreported too long.
func (t *Traffic) Stale(now time.Time) bool {
	return !t.Updated.IsZero() && now.Sub(t.Updated) > trafficStaleAfter
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

// cleanLines is clean for text with line breaks, like a note's template.
func cleanLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = clean(line)
	}
	return strings.Join(lines, "\n")
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
		if from.Traffic.HasLatency && from.Traffic.Latency > total.Latency {
			total.Latency, total.HasLatency = from.Traffic.Latency, true
		}
		if total.Updated.IsZero() || from.Traffic.Updated.Before(total.Updated) {
			total.Updated = from.Traffic.Updated
		}
	}
	return total
}
