package dashboard

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/osteele/liquid"
	"golang.org/x/sync/errgroup"
)

// Annotation surfaces, as the Massdriver UI names them.
const (
	SurfaceCanvasNode   = "instance.canvas_node"
	SurfaceCanvasEdge   = "link.canvas_edge"
	SurfaceCanvasLegend = "environment.canvas_legend"
	SurfaceProject      = "project.header"
)

// Annotation is a note or a pinned widget someone put on a project,
// environment, component, instance, or link. Only its author sees it.
type Annotation struct {
	ID        string           `json:"id"`
	Kind      string           `json:"kind"`
	Surfaces  []string         `json:"surfaces"`
	Template  string           `json:"template"`
	Inputs    Values           `json:"inputs"`
	Entity    AnnotationEntity `json:"entity"`
	ProjectID string           `json:"-"`
}

// AnnotationEntity is what an annotation is on.
type AnnotationEntity struct {
	Typename      string `json:"__typename"`
	ID            string `json:"id"`
	FromComponent *Ref   `json:"fromComponent"`
	ToComponent   *Ref   `json:"toComponent"`
}

const annotationsQuery = `query dashboardAnnotations($organizationId: ID!, $projectId: ID!, $cursor: Cursor) {
  alphaAnnotations(organizationId: $organizationId, filter: {projectId: $projectId}, cursor: $cursor) {
    cursor { next }
    items {
      id kind surfaces template inputs
      entity {
        __typename
        ... on Project { id }
        ... on Environment { id }
        ... on Component { id }
        ... on Instance { id }
        ... on Deployment { id }
        ... on Link { id fromComponent { id } toComponent { id } }
      }
    }
  }
}`

func (a *gqlAPI) Annotations(ctx context.Context, projectID, cursor string) ([]Annotation, string, error) {
	cur := map[string]any{"limit": pageLimit}
	if cursor != "" {
		cur["next"] = cursor
	}
	vars := map[string]any{"organizationId": a.orgID, "projectId": projectID, "cursor": cur}
	var data struct {
		Page page[Annotation] `json:"alphaAnnotations"`
	}
	err := a.request(ctx, "dashboardAnnotations", annotationsQuery, vars, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

// A service account has no annotations, so a failed read leaves them out
// and reports why.
func readAnnotations(ctx context.Context, api API, envs []Environment) ([]Annotation, error) {
	var projects []string
	seen := map[string]bool{}
	for _, env := range envs {
		if env.Project != nil && !seen[env.Project.ID] {
			seen[env.Project.ID] = true
			projects = append(projects, env.Project.ID)
		}
	}
	results := make([][]Annotation, len(projects))
	errs := make([]error, len(projects))
	var group errgroup.Group
	group.SetLimit(stashReads)
	for i, project := range projects {
		group.Go(func() error {
			page := func(ctx context.Context, cursor string) ([]Annotation, string, error) {
				return api.Annotations(ctx, project, cursor)
			}
			results[i], errs[i] = drain(ctx, page)
			for j := range results[i] {
				results[i][j].ProjectID = project
				results[i][j].Template = cleanLines(results[i][j].Template)
			}
			return nil
		})
	}
	_ = group.Wait()
	var all []Annotation
	var err error
	for i := range projects {
		all = append(all, results[i]...)
		if errs[i] != nil {
			err = errs[i]
		}
	}
	return all, err
}

var liquidEngine = liquid.NewEngine()

const maxTemplate = 8192

var (
	markdownImage = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	markdownLink  = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	markdownMarks = regexp.MustCompile(`(^#+\s*)|(\*\*|__|` + "`" + `)`)
)

const (
	noteRenderLimit = 200 * time.Millisecond
	noteCacheSize   = 512
)

// noteCache keeps each note's text for the stash values it was rendered
// with, so a note renders once per change rather than on every read.
var noteCache = struct {
	sync.Mutex
	lines map[string][]string
	slow  map[string]bool
}{lines: map[string][]string{}, slow: map[string]bool{}}

// renderNote renders a note's Liquid template with the stashes it can see
// and reduces the markdown to lines of text. A note renders once for each set
// of stash values, so a template that reads the clock shows the time it last
// rendered. A template that runs past noteRenderLimit is cut off, and is not
// rendered again until its text changes, since Liquid cannot be stopped and
// each try would leave another render running.
func renderNote(template string, bindings map[string]any) []string {
	if len(template) > maxTemplate {
		return []string{"(note too long to show)"}
	}
	noteCache.Lock()
	slow := noteCache.slow[template]
	noteCache.Unlock()
	if slow {
		return []string{tooSlow}
	}
	encoded, err := json.Marshal(bindings)
	key := template + "\x00" + string(encoded)
	if err == nil {
		noteCache.Lock()
		cached, ok := noteCache.lines[key]
		noteCache.Unlock()
		if ok {
			return cached
		}
	}
	done := make(chan []string, 1)
	go func() { done <- renderLines(template, bindings) }()
	var lines []string
	select {
	case lines = <-done:
	case <-time.After(noteRenderLimit):
		noteCache.Lock()
		noteCache.slow[template] = true
		noteCache.Unlock()
		return []string{tooSlow}
	}
	if err == nil {
		noteCache.Lock()
		if len(noteCache.lines) >= noteCacheSize {
			noteCache.lines = map[string][]string{}
		}
		noteCache.lines[key] = lines
		noteCache.Unlock()
	}
	return lines
}

const tooSlow = "(note takes too long to render)"

func renderLines(template string, bindings map[string]any) []string {
	out, err := liquidEngine.ParseAndRenderString(template, bindings)
	if err != nil {
		return []string{"(note does not render: " + clean(firstLine(err.Error())) + ")"}
	}
	out = markdownImage.ReplaceAllString(out, "")
	out = markdownLink.ReplaceAllString(out, "$1")
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(markdownMarks.ReplaceAllString(clean(line), ""))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func (g *Graph) bindingFor(st Stash, now time.Time, kind stash.ScopeKind, id string) map[string]any {
	return map[string]any{"id": id, "stash": st.Values(Scope{Kind: kind, ID: id}, now)}
}

func (g *Graph) applyNote(a Annotation, st Stash, now time.Time) {
	for _, surface := range a.Surfaces {
		switch surface {
		case SurfaceCanvasNode:
			g.noteOnInstances(a, st, now)
		case SurfaceCanvasLegend:
			g.noteOnEnvironment(a, st, now)
		case SurfaceProject:
			g.noteOnProject(a, st, now)
		case SurfaceCanvasEdge:
			g.noteOnLinks(a, st, now)
		}
	}
}

func (g *Graph) noteOnInstances(a Annotation, st Stash, now time.Time) {
	for _, inst := range g.annotatedInstances(a) {
		b := map[string]any{
			"project":     g.bindingFor(st, now, stash.ScopeProject, a.ProjectID),
			"environment": g.bindingFor(st, now, stash.ScopeEnvironment, inst.EnvID),
			"component":   g.bindingFor(st, now, stash.ScopeComponent, inst.ComponentID),
			"instance":    g.bindingFor(st, now, stash.ScopeInstance, inst.ID),
		}
		inst.Notes = append(inst.Notes, renderNote(a.Template, b)...)
	}
}

func (g *Graph) noteOnEnvironment(a Annotation, st Stash, now time.Time) {
	env := g.Env(a.Entity.ID)
	if env == nil || a.Entity.Typename != "Environment" {
		return
	}
	b := map[string]any{
		"project":     g.bindingFor(st, now, stash.ScopeProject, a.ProjectID),
		"environment": g.bindingFor(st, now, stash.ScopeEnvironment, env.ID),
	}
	env.Notes = append(env.Notes, renderNote(a.Template, b)...)
}

func (g *Graph) noteOnProject(a Annotation, st Stash, now time.Time) {
	lines := renderNote(a.Template, map[string]any{"project": g.bindingFor(st, now, stash.ScopeProject, a.ProjectID)})
	for _, env := range g.Envs {
		if env.ProjectID == a.ProjectID {
			env.ProjectNotes = append(env.ProjectNotes, lines...)
		}
	}
}

func (g *Graph) noteOnLinks(a Annotation, st Stash, now time.Time) {
	for _, pair := range g.annotatedLinks(a) {
		b := map[string]any{
			"project":     g.bindingFor(st, now, stash.ScopeProject, a.ProjectID),
			"environment": g.bindingFor(st, now, stash.ScopeEnvironment, pair[0].EnvID),
			"source":      g.bindingFor(st, now, stash.ScopeInstance, pair[0].ID),
			"destination": g.bindingFor(st, now, stash.ScopeInstance, pair[1].ID),
		}
		env := g.Env(pair[0].EnvID)
		for _, line := range renderNote(a.Template, b) {
			env.LinkNotes = append(env.LinkNotes, LinkNote{From: pair[0].ID, To: pair[1].ID, Text: line})
		}
	}
}

func (g *Graph) annotatedInstances(a Annotation) []*InstNode {
	switch a.Entity.Typename {
	case "Instance":
		if inst := g.Instance(a.Entity.ID); inst != nil {
			return []*InstNode{inst}
		}
	case "Component":
		var out []*InstNode
		for _, env := range g.Envs {
			for _, inst := range env.Instances {
				if inst.ComponentID == a.Entity.ID {
					out = append(out, inst)
				}
			}
		}
		return out
	}
	return nil
}

// A link is between components, so it is on an instance pair in every
// environment that runs both.
func (g *Graph) annotatedLinks(a Annotation) [][2]*InstNode {
	e := a.Entity
	if e.Typename != "Link" || e.FromComponent == nil || e.ToComponent == nil {
		return nil
	}
	var out [][2]*InstNode
	for _, env := range g.Envs {
		var from, to *InstNode
		for _, inst := range env.Instances {
			switch inst.ComponentID {
			case e.FromComponent.ID:
				from = inst
			case e.ToComponent.ID:
				to = inst
			}
		}
		if from != nil && to != nil {
			out = append(out, [2]*InstNode{from, to})
		}
	}
	return out
}
