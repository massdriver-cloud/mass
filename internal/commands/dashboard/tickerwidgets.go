package dashboard

import (
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
)

// tickerRoom is how wide one stash change may run on the ticker.
const tickerRoom = 60

const (
	usageBarCells  = 10
	maxHeaderLinks = 8
)

// TickerWidget draws a stash change for the ticker the way the Massdriver UI
// draws its value: as the widget a well-known key or a pinned widget
// annotation makes of it, or else as text. It reads st with the change
// applied, so a pinned widget's other inputs come from the stash.
func TickerWidget(c StashChange, annotations []Annotation, st Stash, now time.Time) string {
	if c.Entry == nil {
		return styleFaint.Render(clip(clean(c.Key), tickerRoom))
	}
	st = withEntry(st, c)
	if w := pinnedWidget(c, annotations, st, now); w != "" {
		return ansi.Truncate(w, tickerRoom, "…")
	}
	if w := wellKnownWidget(c, st, now); w != "" {
		return ansi.Truncate(w, tickerRoom, "…")
	}
	// The key shows only when the value has nothing to say.
	if v := plainValue(*c.Entry); v != "" {
		return styleFaint.Render(clip(v, tickerRoom))
	}
	return styleFaint.Render(clip(clean(c.Key), tickerRoom))
}

func withEntry(st Stash, c StashChange) Stash {
	out := maps.Clone(st)
	if out == nil {
		out = Stash{}
	}
	scope := scopeOf(c.Scope.Kind, c.Scope.ID)
	out[scope] = maps.Clone(st[scope])
	if out[scope] == nil {
		out[scope] = map[string]StashEntry{}
	}
	// Uncleaned, unlike Apply, so line breaks still part words until oneLine
	// folds them into spaces.
	out[scope][c.Key] = *c.Entry
	return out
}

func pinnedWidget(c StashChange, annotations []Annotation, st Stash, now time.Time) string {
	for _, a := range annotations {
		if a.Kind != "WIDGET" || !readsKey(a.Inputs, c.Scope, c.Key) {
			continue
		}
		for _, surface := range a.Surfaces {
			spec, ok := widgetSpecs[surface]
			if !ok {
				continue
			}
			if r, ok := resolve(spec, map[string]any(a.Inputs), st, now); ok {
				if w := drawWidget(surface, Values(r.values)); w != "" {
					return w
				}
			}
		}
	}
	return ""
}

// Newsworthy reports whether a stash change belongs on the ticker, and the
// emoji it rises as: only keys a widget draws, well-known or pinned by an
// annotation, are news. Other keys are a script's own bookkeeping.
func Newsworthy(c StashChange, annotations []Annotation) (string, bool) {
	for _, spec := range widgetSpecs {
		if spec.key == c.Key {
			return StashEmoji(c.Key, c.Entry == nil), true
		}
	}
	for _, a := range annotations {
		if a.Kind != "WIDGET" || !readsKey(a.Inputs, c.Scope, c.Key) {
			continue
		}
		for _, surface := range a.Surfaces {
			if spec, ok := widgetSpecs[surface]; ok {
				return StashEmoji(spec.key, c.Entry == nil), true
			}
		}
	}
	return "", false
}

func readsKey(inputs Values, scope Scope, key string) bool {
	want := scopeOf(scope.Kind, scope.ID)
	for _, in := range inputs {
		src, _ := in.(map[string]any)
		if k, _ := src["key"].(string); k != key {
			continue
		}
		scopeMap, _ := src["scope"].(map[string]any)
		for kind, id := range scopeMap {
			if s, _ := id.(string); scopeOf(stash.ScopeKind(kind), s) == want {
				return true
			}
		}
	}
	return false
}

func wellKnownWidget(c StashChange, st Stash, now time.Time) string {
	for name, spec := range widgetSpecs {
		if spec.key != c.Key {
			continue
		}
		if r, ok := automatic(spec, st, c.Scope, now); ok {
			return drawWidget(name, Values(r.values))
		}
	}
	return ""
}

func drawWidget(surface string, v Values) string {
	switch surface {
	case WidgetHealth:
		state := oneLine(str(v, "state"))
		if state == "" {
			return ""
		}
		return lipgloss.NewStyle().Foreground(healthColor(strings.TrimSpace(state))).Render("● " + state)
	case WidgetUsage:
		value, _ := num(v, "value")
		limit, _ := num(v, "max")
		if limit <= 0 {
			return ""
		}
		u := &Usage{Value: value, Max: limit, Label: oneLine(str(v, "label"))}
		filled := min(max(int(value/limit*usageBarCells+0.5), 0), usageBarCells)
		return lipgloss.NewStyle().Foreground(usageColor(u)).Render(strings.Repeat("▮", filled)) +
			styleFaint.Render(strings.Repeat("▯", usageBarCells-filled)) +
			styleMuted.Render(" "+usageText(u))
	case WidgetTraffic:
		rate, _ := num(v, "rate")
		errs, _ := num(v, "errors")
		out := styleTeal.Render(Rate(rate) + "/s")
		if errs > 0 {
			out += styleFaint.Render(" · ") + lipgloss.NewStyle().Foreground(errorColor(int(errs))).Render("✕"+strconv.Itoa(int(errs)))
		}
		if latency, ok := num(v, "latency"); ok {
			out += styleFaint.Render(" · ") + styleTeal.Render(strconv.FormatFloat(latency, 'f', 0, 64)+"ms")
		}
		return out
	case WidgetBanner:
		msg := oneLine(str(v, "message"))
		if msg == "" {
			return ""
		}
		return lipgloss.NewStyle().Foreground(levelColor(strings.ToLower(strings.TrimSpace(str(v, "level"))))).Render("⚑ " + msg)
	case WidgetLinks:
		labels := linkLabels(v)
		if len(labels) == 0 {
			return ""
		}
		return styleTeal.Render("🔗 " + strings.Join(labels, ", "))
	}
	return ""
}

func linkLabels(v Values) []string {
	links, _ := v["links"].([]any)
	var labels []string
	for _, l := range links {
		link, _ := l.(map[string]any)
		raw, _ := link["url"].(string)
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || !webURL(raw) || u.Host == "" {
			continue
		}
		label, _ := link["label"].(string)
		// The UI's header names an unlabeled link by its host.
		if label = oneLine(label); label == "" {
			label = clean(u.Host)
		}
		labels = append(labels, label)
		if len(labels) == maxHeaderLinks {
			break
		}
	}
	return labels
}

func plainValue(e StashEntry) string {
	if e.Format != string(stash.FormatJSON) {
		return oneLine(text(e))
	}
	if e.Data != nil {
		return summarize(e.Data)
	}
	var v any
	if err := json.Unmarshal([]byte(text(e)), &v); err != nil {
		return oneLine(text(e))
	}
	return summarize(v)
}

func summarize(v any) string {
	switch t := v.(type) {
	case map[string]any:
		keys := slices.Sorted(maps.Keys(t))
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, oneLine(k)+": "+scalar(t[k]))
		}
		return strings.Join(parts, " · ")
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, scalar(item))
		}
		return strings.Join(parts, ", ")
	default:
		return scalar(v)
	}
}

func scalar(v any) string {
	switch t := v.(type) {
	case map[string]any:
		return "{…}"
	case []any:
		return "[…]"
	case string:
		return oneLine(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(t)
		return oneLine(string(b))
	}
}

func oneLine(s string) string {
	return clean(strings.Join(strings.Fields(s), " "))
}
